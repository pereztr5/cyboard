// Package models contains the types for schema 'cyboard'.
package models

import (
	"time"
)

// OtherPoints represents a row from 'cyboard.other_points'.
type OtherPoints struct {
	CreatedAt time.Time `json:"created_at"` // created_at
	TeamID    int       `json:"team_id"`    // team_id
	Points    float32   `json:"points"`     // points
	Reason    string    `json:"reason"`     // reason
}

// Team returns the Team associated with the OtherPoints's TeamID (team_id).
func (os *OtherPoints) Team(db DB) (*Team, error) {
	return TeamByID(db, os.TeamID)
}
