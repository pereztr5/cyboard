package main

import "os/exec"

// BlueteamView has the Team fields needed by the service monitor.
type BlueteamView struct {
	ID         int    `json:"id"`          // id
	Name       string `json:"name"`        // name
	BlueteamIP int16  `json:"blueteam_ip"` // blueteam_ip
}

type MonitorService struct { // `cyboard.service` table
	ID     int      `json:"id"`     // id
	Script string   `json:"script"` // script
	Args   []string `json:"args"`   // args
}

type Check struct {
	TeamID    int
	ServiceID int
	Command   *exec.Cmd
}

type ExitStatus uint16

const (
	ExitStatusPass    = ExitStatus(1)
	ExitStatusFail    = ExitStatus(2)
	ExitStatusPartial = ExitStatus(3)
	ExitStatusTimeout = ExitStatus(4)
)

func (es ExitStatus) String() string {
	var enumVal string

	switch es {
	case ExitStatusPass:
		enumVal = "pass"
	case ExitStatusFail:
		enumVal = "fail"
	case ExitStatusPartial:
		enumVal = "partial"
	case ExitStatusTimeout:
		enumVal = "timeout"
	}

	return enumVal
}

// MarshalText marshals ExitStatus into text.
func (es ExitStatus) MarshalText() ([]byte, error) {
	return []byte(es.String()), nil
}

// ServiceCheck represents a row from 'cyboard.service_check'.
type ServiceCheck struct {
	TeamID    int        `json:"team_id"`    // team_id
	ServiceID int        `json:"service_id"` // service_id
	Status    ExitStatus `json:"status"`     // status
	ExitCode  int16      `json:"exit_code"`  // exit_code
}
