package server

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pereztr5/cyboard/server/models"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	rawCheckCfg    *viper.Viper
	cfgNeedsReload bool
	rando          *rand.Rand

	// dryRun toggles a dummy run of the whole service checker. TODO: Replace with proper tests
	dryRun bool
)

func SetupChecksCfg(v *viper.Viper) {
	rawCheckCfg = v
	dryRun = v.GetBool("dryRun")

	// FIXME(butters): There's an unfortunate race condition in the Viper library.
	//        https://github.com/spf13/viper/issues/174
	// The gist is that there's not synchronization mechanism for this
	// feature, so, if the config gets updated really quickly, the check
	// service would collapse. We could just copy the WatchConfig code
	// and add our own shared file lock as a quick patch.
	rawCheckCfg.OnConfigChange(func(in fsnotify.Event) {
		cfgNeedsReload = true
		Logger.Print(rawCheckCfg.ConfigFileUsed(), " has been updated.")
		Logger.Print("Settings will reload live at the next set of checks.")
	})
	rawCheckCfg.WatchConfig()
}

func prepareChecks(checks []models.Check, scriptsDir string) []models.Check {
	finalChecks := []models.Check{}

	for idx, check := range checks {
		if check.Disable {
			Logger.Warnf("check.%d: DISABLED.", idx)
			continue
		}

		var err error
		check.Script, err = getScript(filepath.Join(scriptsDir, check.Filename))
		if err != nil {
			Logger.Warnf("check.%d: SKIPPING! Failed to locate script: %v", idx, err)
			continue
		}

		finalChecks = append(finalChecks, check)
	}

	Logger.Print("All checks:")
	for i, check := range finalChecks {
		Logger.Printf("  [%d] %v", i, &check)
	}

	return finalChecks
}

func prepareEvent(checkCfg *CheckConfiguration) {
	if time.Now().After(checkCfg.Event.End) {
		if dryRun {
			// Run for 30 seconds if the end time is past already
			checkCfg.Event.End = time.Now().Add(time.Second * 30)
		} else {
			Logger.Error("Event has already ended! " +
				"(Did you forget to update `event_end_time` in the config?)")
		}
	}
	Logger.Print(&checkCfg.Event)
}

func getScript(path string) (*exec.Cmd, error) {
	dir, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	_, err = exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	return exec.Command(dir), nil
}

func score(result models.Result) {
	if !dryRun {
		err := DataAddResult(result, dryRun)
		if err != nil {
			Logger.Error("Could not insert service result:", err)
		}
	} else {
		result.Timestamp = result.Timestamp.Round(time.Millisecond)
		scoreTmplStr := "Timestamp: {{ .Timestamp }} | Group: {{ .Group }} | Team: {{ .Teamname }} | Points: {{ .Points }} | Details: {{ .Details }}\n"
		scoreTmpl := template.Must(template.New("result").Parse(scoreTmplStr))
		err := scoreTmpl.Execute(Logger.Out, result)
		if err != nil {
			Logger.Error("Executing template:", err)
		}
	}
}

func scoreAll(results []models.Result) {
	if !dryRun {
		err := DataAddResults(results, dryRun)
		if err != nil {
			Logger.Error("Could not insert service result:", err)
		}
	} else {
		for _, result := range results {
			score(result)
		}
	}
}

func startCheckService(checkCfg *CheckConfiguration, teams []models.Team) {
	event := &checkCfg.Event
	checks := checkCfg.Checks
	status := make(chan models.Result)
	resultsBuf := make([]models.Result, len(teams)*len(checks))

	// Run command every x seconds until scheduled end time
	Logger.Println("Starting Checks")
	checkTicker := time.NewTicker(event.Intervals)
	waitingOnReload := false
	freeTime := event.Intervals - event.Timeout
	if freeTime <= 0 {
		// There must be some jitter
		freeTime = 1
	}

	for {
		now := time.Now()

		// Add unpredictability to the service checking by waiting some time
		jitter := time.Duration(rando.Int63n(int64(freeTime)))
		<-time.After(jitter)

		if !cfgNeedsReload && now.Before(event.End) {

			// When there's nothing to do, log once, then just keep
			// waiting for the config to be reloaded, or the event to end.
			if event.OnBreak || len(checks) == 0 {
				if !waitingOnReload {
					if len(checks) == 0 {
						Logger.Error("No checks enabled/configured in: ", rawCheckCfg.ConfigFileUsed())
						Logger.Error("Waiting for config file to be updated...")
					} else {
						Logger.Warn("We're on break! Enjoy it! (Then update the config, setting `on_break = false`)")
					}

					waitingOnReload = true
				}

				<-checkTicker.C
				continue
			}

			Logger.Printf("Running Checks. Started +jitter = %s +%v",
				now.Format(time.RFC3339), jitter.Truncate(time.Millisecond))
			for _, team := range teams {
				for _, check := range checks {
					go runCmd(team, check, now, event.Timeout, status)
				}
			}
			for idx := range resultsBuf {
				resultsBuf[idx] = <-status
			}
			scoreAll(resultsBuf)
		} else {
			checkTicker.Stop()
			if now.After(event.End) {
				Logger.Println("Done Checking Services")
			}
			break
		}
		<-checkTicker.C
	}
}

func runCmd(team models.Team, check models.Check, timestamp time.Time, timeout time.Duration, status chan models.Result) {
	// TODO: Currently only one IP per team is supported
	cmd := *check.Script
	cmd.Args = parseArgs(cmd.Path, check.Args, team.Ip)

	var out bytes.Buffer
	if dryRun {
		cmd.Stdout, cmd.Stderr = &out, &out
	}

	err := cmd.Start()

	result := models.Result{
		Type:       "Service",
		Timestamp:  timestamp,
		Group:      check.Name,
		Teamname:   team.Name,
		Teamnumber: team.Number,
	}

	if err != nil {
		Logger.Error("Could not run script:", err)
		result.Details = "Status: 127" // 127=command not found: http://www.tldp.org/LDP/abs/html/exitcodes.html
		status <- result
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			//TODO: If it cannot kill it what to do we do?
			Logger.Error("Failed to Kill:", err)
		}
		result.Details = "Status: timed out"
	case <-done:
		// As long as it is done the error doesn't matter
		exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()

		if !dryRun {
			result.Details = fmt.Sprintf("Status: %d", exitCode)
		} else {
			result.Details = fmt.Sprintf("%s\n%s", strings.Join(cmd.Args, " "), out.Bytes())
		}

		if exitCode >= len(check.Points) {
			Logger.Warnf("Unexpected exit code (will be awarded '0' points): exitCode=%d, checkName=%s", exitCode, check.Name)
		} else {
			result.Points = check.Points[exitCode]
		}
	}
	status <- result
}

func parseArgs(name string, args string, ip string) []string {
	//TODO: Quick fix but need to com back and do this right
	//TODO(pereztr): This should at least have to be surrounded in braces or some meta-chars
	const ReplacementText = "IP"
	nArgs := name + " " + strings.Replace(args, ReplacementText, ip, 1)
	return strings.Split(nArgs, " ")
}

func testData() []models.Team {
	var teams []models.Team
	for i := 0; i < 2; i++ {
		t := models.Team{
			Group:  "TEST",
			Number: 90 + i,
			Name:   "team9" + strconv.Itoa(i),
			Ip:     "127.0.0.1",
		}
		teams = append(teams, t)
	}
	return teams
}

func ChecksRun(checkCfg *CheckConfiguration) {
	SetupCheckServiceLogger(&checkCfg.Log)
	SetupMongo(&checkCfg.Database, nil)
	rando = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	for {
		cfgNeedsReload = false
		prepareEvent(checkCfg)
		checkCfg.Checks = prepareChecks(checkCfg.Checks, checkCfg.Event.ChecksDir)
		if !dryRun {
			teams, err := DataGetTeamIps()
			if err != nil {
				Logger.Fatal("Could not get teams for service checks: ", err)
			}
			startCheckService(checkCfg, teams)
		} else {
			teams := testData()
			startCheckService(checkCfg, teams)
		}

		// If the checker service stopped other than by a config reload, then the event has reached
		// it's end and it's time to shut down.
		// Else, reload the config and start again!
		if !cfgNeedsReload {
			break
		} else {
			c := &CheckConfiguration{}
			if err := rawCheckCfg.Unmarshal(c); err != nil {
				Logger.Warnf("Unable to update config:", err)
				Logger.Warn("Config was not refreshed, but checking will restart!")
			} else {
				checkCfg = c
			}
		}
	}
}
