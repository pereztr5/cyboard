// Package models contains the types for schema 'cyboard'.
package models

import (
	"time"
)

// CtfSolve represents a row from 'cyboard.ctf_solve'.
type CtfSolve struct {
	CreatedAt   time.Time `json:"created_at"`   // created_at
	TeamID      int       `json:"team_id"`      // team_id
	ChallengeID int       `json:"challenge_id"` // challenge_id
}

// Challenge returns the Challenge associated with the CtfSolve's ChallengeID (challenge_id).
func (cs *CtfSolve) Challenge(db DB) (*Challenge, error) {
	return ChallengeByID(db, cs.ChallengeID)
}

// Team returns the Team associated with the CtfSolve's TeamID (team_id).
func (cs *CtfSolve) Team(db DB) (*Team, error) {
	return TeamByID(db, cs.TeamID)
}
