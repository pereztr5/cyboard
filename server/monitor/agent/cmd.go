package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func getScript(path string, args []string) (*exec.Cmd, error) {
	execpath, err := exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	abspath, err := filepath.Abs(execpath)
	if err != nil {
		return nil, err
	}
	return exec.Command(abspath, args...), nil
}

func prepareChecks(services []MonitorService, team *BlueteamView, scriptsDir string) ([]Check, error) {
	checks := []Check{}

	// argCache saves a few cpu cycles doing the same argument variable substitution
	argCache := map[string]string{}

	for i := range services {
		srv := &services[i]

		args := make([]string, 0, len(srv.Args))

		// Perform variable exansion
		for _, arg := range srv.Args {
			s, ok := argCache[arg]
			if !ok {
				s = arg
				if strings.IndexByte(s, '{') != -1 {
					teamIDstr := strconv.FormatInt(int64(team.ID), 10)
					teamSigIPOctet := strconv.FormatInt(int64(team.BlueteamIP), 10)
					s = strings.Replace(arg, "{t}", teamSigIPOctet, -1)
					s = strings.Replace(s, "{TEAM_NAME}", team.Name, -1)
					s = strings.Replace(s, "{TEAM_ID}", teamIDstr, -1)
				}
				argCache[arg] = s
			}
			args = append(args, s)
		}

		path := filepath.Join(scriptsDir, srv.Script)
		script, err := getScript(path, args)
		if err != nil {
			return nil, fmt.Errorf("failed to init check: %v (team=%q, service=%q)",
				err, team.Name, srv.Script)
		}
		checks = append(checks, Check{TeamID: team.ID, ServiceID: srv.ID, Command: script})
	}

	// log.Print("All checks:")
	// for _, chk := range checks {
	//     log.Printf(`  [%d] Check{fullcmd="%s %s"}`,
	//         chk.ServiceID, filepath.Base(chk.Command.Path), strings.Join(chk.Command.Args, " "))
	// }

	return checks, nil
}

func getCmdResult(cmd *exec.Cmd, timeout time.Duration) (int16, ExitStatus) {
	var code int16
	var status ExitStatus

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("failed to kill: %v (script=%q)", err, filepath.Base(cmd.Path))
		}
		code, status = 129, ExitStatusTimeout
	case <-done:
		// As long as it is done the error doesn't matter
		code = int16(cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus())

		switch code {
		case 0:
			status = ExitStatusPass
		case 1:
			status = ExitStatusPartial
		default:
			status = ExitStatusFail
		}
	}
	return code, status
}

func runCmd(check *Check, timeout time.Duration) ServiceCheck {
	cmd := *check.Command

	result := ServiceCheck{
		TeamID:    check.TeamID,
		ServiceID: check.ServiceID,
	}

	if err := cmd.Start(); err != nil {
		log.Println("Could not run script:", err)
		result.ExitCode = 127 // 127=command not found: http://www.tldp.org/LDP/abs/html/exitcodes.html
		result.Status = ExitStatusTimeout
		return result
	}

	result.ExitCode, result.Status = getCmdResult(&cmd, timeout)
	return result
}
