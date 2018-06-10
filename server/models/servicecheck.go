// Package models contains the types for schema 'cyboard'.
package models

import (
	"time"
)

// ServiceCheck represents a row from 'cyboard.service_check'.
type ServiceCheck struct {
	CreatedAt time.Time  `json:"created_at"` // created_at
	TeamID    int        `json:"team_id"`    // team_id
	ServiceID int        `json:"service_id"` // service_id
	Status    ExitStatus `json:"status"`     // status
	ExitCode  int16      `json:"exit_code"`  // exit_code
}

// Service returns the Service associated with the ServiceCheck's ServiceID (service_id).
func (sc *ServiceCheck) Service(db DB) (*Service, error) {
	return ServiceByID(db, sc.ServiceID)
}

// Team returns the Team associated with the ServiceCheck's TeamID (team_id).
func (sc *ServiceCheck) Team(db DB) (*Team, error) {
	return TeamByID(db, sc.TeamID)
}
