package models

import (
	"time"
)

// OtherPoints represents a row from 'cyboard.other_points'.
// This type is not directly used, but left here as a reference.
type OtherPoints struct {
	CreatedAt time.Time `json:"created_at"` // created_at
	TeamID    int       `json:"team_id"`    // team_id
	Points    float32   `json:"points"`     // points
	Reason    string    `json:"reason"`     // reason
}

// Insert a bonus point award/deduction into the database.
func (op *OtherPoints) Insert(db DB) error {
	const sqlstr = `INSERT INTO other_points (team_id, points, reason) VALUES ($1, $2, $3)`
	_, err := db.Exec(sqlstr, op.TeamID, op.Points, op.Reason)
	return err
}

// OtherPointsSlice is an array of bonus points, suitable to insert many of at once.
type OtherPointsSlice []OtherPoints

// Insert many bonus point scores into the database at once.
func (ops OtherPointsSlice) Insert(db DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const sqlstr = `INSERT INTO other_points (created_at, team_id, points, reason) VALUES ($1, $2, $3, $4)`
	for _, op := range ops {
		_, err := tx.Exec(sqlstr, op.CreatedAt, op.TeamID, op.Points, op.Reason)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
