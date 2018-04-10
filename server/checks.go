package server

import (
	"bytes"
	"fmt"
	"html/template"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type EventSettings struct {
	ChecksDir string    `mapstructure:"checks_dir"`
	End       time.Time `mapstructure:"event_end_time"`
	Intervals time.Duration
	Timeout   time.Duration
	OnBreak   bool `mapstructure:"on_break"`
}

type Check struct {
	Name     string `mapstructure:"check_name"`
	Filename string
	Script   *exec.Cmd
	Args     string
	Points   []int
	Disable  bool
}

type CheckConfiguration struct {
	Event    EventSettings
	Log      LogSettings
	Database DBSettings
	Checks   []Check
}

var (
	rawCheckCfg    *viper.Viper
	cfgNeedsReload bool

	// dryRun toggles a dummy run of the whole service checker. TODO: Replace with proper tests
	dryRun bool
)

func (c *Check) String() string {
	return fmt.Sprintf(`Check{name=%q, fullcmd="%s %s", pts=%v}`,
		c.Name, filepath.Base(c.Script.Path), c.Args, c.Points)
}

func (es *EventSettings) String() string {
	return fmt.Sprintf(`Event{end=%v, interval=%v, timeout=%v, OnBreak=%v}`,
		es.End.Format(time.UnixDate), es.Intervals, es.Timeout, es.OnBreak)
}

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

func prepareChecks(checks []Check, scriptsDir string) []Check {
	finalChecks := []Check{}

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

func score(result Result) {
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

func scoreAll(results []Result) {
	if !dryRun {
		err := DataAddResults(results, dryRun)
		if err != nil {
			Logger.Error("Could not insert service result: ", err)
		}
		if err = PostgresScoreServices(results); err != nil {
			Logger.Error("Could not insert service results in postgres: ", err)
		}
	} else {
		for _, result := range results {
			score(result)
		}
	}
}

func startCheckService(checkCfg *CheckConfiguration, teams []Team) {
	event := &checkCfg.Event
	checks := checkCfg.Checks
	status := make(chan Result)
	resultsBuf := make([]Result, len(teams)*len(checks))

	// Run command every x seconds until scheduled end time
	Logger.Println("Starting Checks")
	checkTicker := time.NewTicker(event.Intervals)
	waitingOnReload := false

	for {
		now := time.Now()
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

			Logger.Println("Running Checks")
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

func runCmd(team Team, check Check, timestamp time.Time, timeout time.Duration, status chan Result) {
	// TODO: Currently only one IP per team is supported
	cmd := *check.Script
	cmd.Args = parseArgs(cmd.Path, check.Args, team.Ip)

	var out bytes.Buffer
	if dryRun {
		cmd.Stdout, cmd.Stderr = &out, &out
	}

	err := cmd.Start()

	result := Result{
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

func testData() []Team {
	var teams []Team
	for i := 0; i < 2; i++ {
		t := Team{
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
	SetupPostgres(checkCfg.Database.PostgresURI)

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
