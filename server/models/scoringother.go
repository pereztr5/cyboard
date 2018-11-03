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

// Insert a bonus point award/deduction into the database.
func (op *OtherPoints) Insert(db DB) error {
	const sqlstr = `INSERT INTO other_points (team_id, points, reason) VALUES ($1, $2, $3)`
	_, err := db.Exec(sqlstr, op.TeamID, op.Points, op.Reason)
	return err
}

// OtherPointsView groups together a row of other_points by timestamp
type OtherPointsView struct {
	CreatedAt time.Time `json:"created_at"` // created_at
	Teams     []string  `json:"teams"`      // array of team names
	Points    float32   `json:"points"`     // points
	Reason    string    `json:"reason"`     // reason
}

// AllBonusPoints returns all the bonus point totals, grouped by timestamp.
func AllBonusPoints(db DB) ([]OtherPointsView, error) {
	const sqlstr = `SELECT created_at, ARRAY_AGG(t.name), points, reason
	FROM other_points JOIN blueteam AS t ON team_id = t.id
	GROUP BY created_at, points, reason`
	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []OtherPointsView{}
	for rows.Next() {
		x := OtherPointsView{}
		if err = rows.Scan(&x.CreatedAt, &x.Teams, &x.Points, &x.Reason); err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}

// OtherPointsSlice is an array of bonus points, suitable to insert many of at once.
type OtherPointsSlice []OtherPoints

// Insert many bonus point scores into the database at once.
// The incoming slice should have the CreatedAt field set to the same value on each
// struct, allowing a batch of bonus points to 'come in' at exactly the same time.
func (ops OtherPointsSlice) Insert(db TXer) error {
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
