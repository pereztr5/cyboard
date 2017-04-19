package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Check struct {
	Name   string
	Script *exec.Cmd
	Args   string
	Points map[int]int
}

type EventSettings struct {
	Start     time.Time
	End       time.Time
	Timeout   time.Duration
	Intervals time.Duration
	OnBreak   bool
}

var (
	Event          EventSettings
	checkcfg       *viper.Viper
	cfgNeedsReload bool
	dryRun         bool
)

func (c *Check) String() string {
	return fmt.Sprintf(`Check{name=%q, fullcmd="%s %s", pts=%d}`,
		c.Name, filepath.Base(c.Script.Path), c.Args, c.Points[0])
}

func (es *EventSettings) String() string {
	return fmt.Sprintf(`Event{end=%v, interval=%v, timeout=%v, OnBreak=%v}`,
		es.End.Format(time.UnixDate), es.Intervals, es.Timeout, es.OnBreak)
}

func SetupCfg(cfg *viper.Viper, dryRunBool bool) {
	checkcfg = cfg
	dryRun = dryRunBool

	// FIXME(butters): There's an unfortunate race condition in the Viper library.
	//        https://github.com/spf13/viper/issues/174
	// The gist is that there's not synchronization mechanism for this
	// feature, so, if the config gets updated really quickly, the check
	// service would collapse. We could just copy the WatchConfig code
	// and add our own shared file lock as a quick patch.
	checkcfg.WatchConfig()
	checkcfg.OnConfigChange(func(in fsnotify.Event) {
		cfgNeedsReload = true
		Logger.Println(checkcfg.ConfigFileUsed(), "has been updated. ")
		Logger.Println("Settings will reload live at the next set of checks.")
	})
}

func getChecks() (checks []Check) {
	checksDir := checkcfg.GetString("checks_dir")
	for n := range checkcfg.GetStringMap("checks") {
		checkKey := "checks." + n
		readCfgString := func(s string) string {
			return checkcfg.GetString(checkKey + "." + s)
		}

		if checkcfg.GetBool(checkKey + ".disable") {
			Logger.Printf("%v: DISABLED.", checkKey)
			continue
		}

		script, err := getScript(filepath.Join(checksDir, readCfgString("filename")))
		if err != nil {
			Logger.Printf("%v: SKIPPING! Failed to locate script: %v", checkKey, err)
			continue
		}
		pts, err := getPoints(checkKey + ".points")
		if err != nil {
			Logger.Printf("%v: SKIPPING! Failed to get point totals: %v", checkKey, err)
			continue
		}
		s := Check{
			Name:   readCfgString("check_name"),
			Script: script,
			Args:   readCfgString("args"),
			Points: pts,
		}
		checks = append(checks, s)
	}
	Logger.Println("All checks:")
	for i, chk := range checks {
		Logger.Printf("  [%d] %v\n", i, &chk)
	}

	return checks
}

func loadSettings() {
	// Get Event details
	var err error
	Event.End, err = time.Parse(time.UnixDate, checkcfg.GetString("event_end_time"))
	if err != nil {
		Logger.Fatal("Failed to parse event_end_time:", err)
	}

	if time.Now().After(Event.End) {
		if dryRun {
			// Run for 30 seconds if the end time is past already
			Event.End = time.Now().Add(time.Second * 30)
		} else {
			Logger.Println("Event has already ended! " +
				"(Did you forget to update `event_end_time` in the config?)")
		}
	}
	Event.Intervals = checkcfg.GetDuration("intervals")
	Event.Timeout = checkcfg.GetDuration("timeout")
	Event.OnBreak = checkcfg.GetBool("on_break")
	Logger.Println(&Event)
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

func getPoints(ptsKey string) (map[int]int, error) {
	var p []int
	err := checkcfg.UnmarshalKey(ptsKey, &p)
	if err != nil {
		return nil, err
	}
	points := make(map[int]int)
	for i, v := range p {
		points[i] = v
	}
	return points, nil
}

func score(result Result) {
	//fmt.Printf("%s %s\nService: %s\tStatus: %d\n%v\n", res.Team.Teamname, res.Team.Ips[0], res.Service, res.Status, res.Output)
	if !dryRun {
		err := DataAddResult(result, dryRun)
		if err != nil {
			Logger.Println("Could not insert service result:", err)
		}
	} else {
		result.Timestamp = result.Timestamp.Round(time.Millisecond)
		scoreTmplStr := "Timestamp: {{ .Timestamp }} | Group: {{ .Group }} | Team: {{ .Teamname }} | Details: {{ .Details }} | Points: {{ .Points }}\n"
		scoreTmpl := template.Must(template.New("result").Parse(scoreTmplStr))
		err := scoreTmpl.Execute(os.Stdout, result)
		if err != nil {
			Logger.Println("executing template:", err)
		}
	}
}

func startCheckService(teams []Team, checks []Check) {
	Event.Start = time.Now()
	//bio := bufio.NewReader(os.Stdin)
	//Logger.Print("Press enter to start....")
	//_, _ = bio.ReadString('\n')

	// Run command every x seconds until scheduled end time
	Logger.Println("Starting Checks")
	checkTicker := time.NewTicker(Event.Intervals)
	status := make(chan Result)
	waitingOnReload := false
	for {
		now := time.Now()
		if !cfgNeedsReload && now.Before(Event.End) {

			// When there's nothing to do, log once, then just keep
			// waiting for the config to be reloaded, or the event to end.
			if Event.OnBreak || len(checks) == 0 {
				if !waitingOnReload {
					if len(checks) == 0 {
						Logger.Println("No checks enabled/configured in: ", checkcfg.ConfigFileUsed())
					} else {
						Logger.Println("We're on break! Enjoy it! (Then update the config, setting `on_break = false`)")
					}

					waitingOnReload = true
				}

				<-checkTicker.C
				continue
			}

			Logger.Println("Running Checks")
			for _, team := range teams {
				for _, check := range checks {
					go runCmd(team, check, now, status)
				}
			}
			amtChecks := len(teams) * len(checks)
			for j := 0; j < amtChecks; j++ {
				select {
				case res := <-status:
					go score(res)
				}
			}
		} else {
			checkTicker.Stop()
			if now.After(Event.End) {
				Logger.Println("Done Checking Services")
			}
			break
		}
		<-checkTicker.C
	}
}

func runCmd(team Team, check Check, timestamp time.Time, status chan Result) {
	// TODO: Currently only one IP per team is supported
	cmd := *check.Script
	cmd.Args = parseArgs(cmd.Path, check.Args, team.Ip)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Start()
	if err != nil {
		Logger.Println("Could not run script:", err)
	} else {
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		result := Result{
			Type:       "Service",
			Timestamp:  timestamp,
			Group:      check.Name,
			Teamname:   team.Name,
			Teamnumber: team.Number,
		}
		select {
		case <-time.After(Event.Timeout):
			if err := cmd.Process.Kill(); err != nil {
				//TODO: If it cannot kill it what to do we do?
				// If fatal then everything stops
				Logger.Println("Failed to Kill:", err)
			}
			//Logger.Println(check.Name, "timed out")
			result.Details = "Status: timed out"
			status <- result
		case _ = <-done:
			// As long as it is done the error doesn't matter
			s := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			var detail string
			if !dryRun {
				detail = fmt.Sprintf("Status: %d", s)
			} else {
				detail = stdout.String() + "\t" + strings.Join(cmd.Args, " ")
			}
			result.Details = detail
			result.Points = check.Points[s]
			status <- result
		}
	}
}

func parseArgs(name string, args string, ip string) []string {
	//TODO: Quick fix but need to com back and do this right
	//TODO(pereztr): This should at least have to be surrounded in braces or some meta-chars
	const ReplacementText = "IP"
	nArgs := name + " " + strings.Replace(args, ReplacementText, ip, 1)
	return strings.Split(nArgs, " ")
}

func getOutput(stdout io.ReadCloser) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	return buf.String()
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

func ChecksRun() {
	SetupCheckServiceLogger(checkcfg)

	for {
		cfgNeedsReload = false
		loadSettings()
		checks := getChecks()
		if !dryRun {
			teams, err := DataGetTeamIps()
			if err != nil {
				Logger.Fatalln("Could not get teams for service checks:", err)
			}
			startCheckService(teams, checks)
		} else {
			teams := testData()
			startCheckService(teams, checks)
		}
		if !cfgNeedsReload {
			break
		}
	}
}
