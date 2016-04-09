package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

type Check struct {
	Name   string
	Script exec.Cmd
	Points map[int]int
}

type EventSettings struct {
	Start     time.Time
	End       time.Time
	Timeout   time.Duration
	Intervals time.Duration
}

var Event EventSettings
var cfgCheck *viper.Viper

func getConfig() {
	cfgCheck = viper.New()
	cfgCheck.SetConfigName("checks")
	cfgCheck.AddConfigPath(".")
	err := cfgCheck.ReadInConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// Get Event details
	Event.End, err = time.Parse(time.UnixDate, cfgCheck.GetString("event_end_time"))
	if err != nil {
		log.Fatal(err)
	}

	// Run for 30 seconds if the end time is past already
	if time.Now().After(Event.End) {
		Event.End = time.Now().Add(time.Second * 30)
	}
	Event.Intervals = cfgCheck.GetDuration("intervals")
	Event.Timeout = cfgCheck.GetDuration("timeout")
}

func getChecks() (checks []Check) {
	checksDir := cfgCheck.GetString("checks_dir")
	for n := range cfgCheck.GetStringMap("checks") {
		check := "checks." + n
		s := Check{
			Name:   cfgCheck.GetString(check + ".check_name"),
			Script: getScript(checksDir + "/" + cfgCheck.GetString(check+".filename")),
			Points: getPoints(check + ".points"),
		}
		// Get Arguments
		s.Script.Args = append(s.Script.Args, getArgs(cfgCheck.GetString(check+".args"))...)
		checks = append(checks, s)
	}
	return checks
}

func start(teams []Team, checks []Check) {
	Event.Start = time.Now()
	checkTicker := time.NewTicker(Event.Intervals)

	status := make(chan CheckResult)

	// Run command every x seconds until scheduled end time
	for t := range checkTicker.C {
		if time.Now().Before(Event.End) {
			fmt.Printf("%s Running Checks\n", t)
			for _, team := range teams {
				for _, check := range checks {
					go runCmd(team, check, status)
				}
			}
			amtChecks := len(teams) * len(checks)
			for j := 0; j < amtChecks; j++ {
				select {
				case res := <-status:
					fmt.Printf("%s %s\nService: %s\tStatus: %d\n%v\n", res.Team.Teamname, res.Team.Ips[0], res.Service, res.Status, res.Output)
				}
			}
		} else {
			checkTicker.Stop()
			fmt.Println("Done Checking Services")
			break
		}
	}
}

func score() {
	// Get Teamname and status
	// Based on status will insert points to team
}

func runCmd(team Team, check Check, status chan CheckResult) {
	for _, ip := range team.Ips {
		go func() {
			cmd := &check.Script
			cmd.Args = append(cmd.Args, ip)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Start()
			if err != nil {
				log.Printf("Could not run script: %s\n", err)
			} else {
				done := make(chan error, 1)
				go func() {
					done <- cmd.Wait()
				}()
				select {
				case <-time.After(Event.Timeout):
					if err := cmd.Process.Kill(); err != nil {
						//TODO: If it cannot kill it what to do we do?
						// If fatal then everything stops
						log.Printf("Failed to Kill: %v\n", err)
					}
					log.Printf("%s timed out\n", check.Name)
					// Send status with timeout
				case _ = <-done:
					// As long as it is done what the error doesn't matter
					status <- CheckResult{
						Team:    team,
						Service: check.Name,
						Status:  cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus(),
						Output:  out.String(),
					}
				}
			}
		}()
	}
}

func getScript(path string) exec.Cmd {
	dir, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	return *exec.Command(dir)
}

func getArgs(args string) []string {
	return strings.Split(args, " ")
}

func getPoints(name string) map[int]int {
	var p []int
	err := cfgCheck.UnmarshalKey(name, &p)
	if err != nil {
		fmt.Println(err)
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
