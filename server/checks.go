package server

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Check struct {
	Name   string
	Script exec.Cmd
	Args   string
	Points map[int]int
}

type EventSettings struct {
	Start     time.Time
	End       time.Time
	Timeout   time.Duration
	Intervals time.Duration
}

var CheckCmd = &cobra.Command{
	Use:   "checks",
	Short: "Run Service Checks",
	Long:  `Will get config file for checks and then running it at intervals`,
	Run:   checksRun,
}

var (
	Event    EventSettings
	cfgCheck string
	checkcfg *viper.Viper
	dryRun   bool
)

func init() {
	cobra.OnInitialize(initCheckConfig)
	CheckCmd.PersistentFlags().StringVar(&cfgCheck, "config", "", "service check config file (default is $HOME/.cyboard/checks.toml)")
	CheckCmd.Flags().BoolVarP(&dryRun, "dry", "", false, "Do a dry run of checks")
}

func initCheckConfig() {
	checkcfg = viper.New()
	if cfgCheck != "" {
		checkcfg.SetConfigFile(cfgCheck)
	}
	checkcfg.SetConfigName("checks")
	checkcfg.AddConfigPath("$HOME/.cyboard/")
	checkcfg.AddConfigPath(".")
	err := checkcfg.ReadInConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

func loadSettings() {
	// Get Event details
	var err error
	Event.End, err = time.Parse(time.UnixDate, checkcfg.GetString("event_end_time"))
	if err != nil {
		Logger.Fatal(err)
	}

	// Run for 30 seconds if the end time is past already
	if time.Now().After(Event.End) {
		Event.End = time.Now().Add(time.Second * 30)
	}
	Event.Intervals = checkcfg.GetDuration("intervals")
	Event.Timeout = checkcfg.GetDuration("timeout")
}

func getChecks() (checks []Check) {
	checksDir := checkcfg.GetString("checks_dir")
	for n := range checkcfg.GetStringMap("checks") {
		check := "checks." + n
		s := Check{
			Name:   checkcfg.GetString(check + ".check_name"),
			Script: getScript(checksDir + "/" + checkcfg.GetString(check+".filename")),
			Args:   checkcfg.GetString(check + ".args"),
			Points: getPoints(check + ".points"),
		}
		checks = append(checks, s)
	}
	return checks
}

func score(result Result) {
	//fmt.Printf("%s %s\nService: %s\tStatus: %d\n%v\n", res.Team.Teamname, res.Team.Ips[0], res.Service, res.Status, res.Output)
	if !dryRun {
		err := DataAddResult(result, dryRun)
		if err != nil {
			Logger.Printf("Could not insert service result: %v\n", err)
		}
	} else {
		go func() {
			tmpl := `
Group: {{ .Group }}
Team: {{ .Teamname }}
Details: {{ .Details }}
Points: {{ .Points }}
`
			t, err := template.New("result").Parse(tmpl)
			if err != nil {
				Logger.Fatal(err)
			}
			err = t.Execute(os.Stdout, result)
			if err != nil {
				Logger.Println("executing template:", err)
			}
		}()
	}
}

func start(teams []Team, checks []Check) {
	Event.Start = time.Now()
	checkTicker := time.NewTicker(Event.Intervals)
	status := make(chan Result)
	bio := bufio.NewReader(os.Stdin)
	fmt.Printf("Press enter to start....")
	_, _ = bio.ReadString('\n')
	// Run command every x seconds until scheduled end time
	fmt.Printf("%v: Starting Checks\n", time.Now())
	for t := range checkTicker.C {
		if time.Now().Before(Event.End) {
			fmt.Printf("%v: Running Checks\n", t)
			for _, team := range teams {
				for _, check := range checks {
					go runCmd(team, check, status)
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
			fmt.Printf("%v: Done Checking Services\n", t)
			break
		}
	}
}

func runCmd(team Team, check Check, status chan Result) {
	// TODO: Currently only one IP per team is supported
	cmd := &check.Script
	cmd.Args = parseArgs(cmd.Path, check.Args, team.Ip)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Start()
	if err != nil {
		Logger.Printf("Could not run script: %s\n", err)
	} else {
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		result := Result{
			Type:       "Service",
			Group:      check.Name,
			Teamname:   team.Name,
			Teamnumber: team.Number,
		}
		select {
		case <-time.After(Event.Timeout):
			if err := cmd.Process.Kill(); err != nil {
				//TODO: If it cannot kill it what to do we do?
				// If fatal then everything stops
				Logger.Printf("Failed to Kill: %v\n", err)
			}
			Logger.Printf("%s timed out\n", check.Name)
			result.Details = "Status: " + "timed out"
			status <- result
		case _ = <-done:
			// As long as it is done the error doesn't matter
			s := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			var detail string
			if !dryRun {
				detail = "Status: " + strconv.Itoa(s)
			} else {
				detail = stdout.String() + "\t" + strings.Join(cmd.Args, " ")
			}
			result.Details = detail
			result.Points = check.Points[s]
			status <- result
		}
	}
}

func getScript(path string) exec.Cmd {
	dir, err := filepath.Abs(path)
	if err != nil {
		Logger.Fatal(err)
	}
	return *exec.Command(dir)
}

func parseArgs(name string, args string, ip string) []string {
	//TODO: Quick fix but need to com back and do this right
	nArgs := name + " " + strings.Replace(args, "IP", ip, 1)
	return strings.Split(nArgs, " ")
}

func getPoints(name string) map[int]int {
	var p []int
	err := checkcfg.UnmarshalKey(name, &p)
	if err != nil {
		Logger.Println(err)
	}
	points := make(map[int]int)
	for i, v := range p {
		points[i] = v
	}
	return points
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

func checksRun(cmd *cobra.Command, args []string) {
	if !dryRun {
		loadSettings()
		checks := getChecks()
		teams, err := DataGetTeamIps()
		if err != nil {
			Logger.Fatalf("Could not get teams for service checks: %v\n", err)
		}
		start(teams, checks)
	} else {
		loadSettings()
		checks := getChecks()
		teams := testData()
		start(teams, checks)
	}
}
