// Package models contains the types for schema 'cyboard'.
package models

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Challenge represents a row from 'cyboard.challenge'.
type Challenge struct {
	ID       int     `json:"id"`       // id
	Name     string  `json:"name"`     // name
	Category string  `json:"category"` // category
	Designer string  `json:"designer"` // designer
	Flag     string  `json:"flag"`     // flag
	Total    float32 `json:"total"`    // total
	Body     string  `json:"body"`     // body
	Hidden   bool    `json:"hidden"`   // hidden

	CreatedAt  time.Time `json:"created_at"`  // created_at
	ModifiedAt time.Time `json:"modified_at"` // modified_at
}

// Insert inserts the Challenge to the database.
func (c *Challenge) Insert(db DB) error {
	const sqlstr = `INSERT INTO cyboard.challenge (` +
		`name, category, designer, flag, total, body, hidden` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7` +
		`) RETURNING id`

	return db.QueryRow(sqlstr, c.Name, c.Category, c.Designer, c.Flag, c.Total, c.Body, c.Hidden).Scan(&c.ID)
}

// Update updates the Challenge in the database.
func (c *Challenge) Update(db DB) error {
	const sqlstr = `UPDATE cyboard.challenge SET (` +
		`name, category, designer, flag, total, body, hidden` +
		`) = ( ` +
		`$2, $3, $4, $5, $6, $7, $8` +
		`) WHERE id = $1`

	_, err := db.Exec(sqlstr, c.ID, c.Name, c.Category, c.Designer, c.Flag, c.Total, c.Body, c.Hidden)
	return err
}

// Delete deletes the Challenge from the database.
func (c *Challenge) Delete(db DB) error {
	const sqlstr = `DELETE FROM cyboard.challenge WHERE id = $1`

	_, err := db.Exec(sqlstr, c.ID)
	return err
}

// ChallengeByFlag retrieves a row from 'cyboard.challenge' as a Challenge.
func ChallengeByFlag(db DB, flag string) (*Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, designer, flag, total, body, hidden, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE flag = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, flag).Scan(&c.ID, &c.Name, &c.Category, &c.Designer, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChallengeByName retrieves a row from 'cyboard.challenge' as a Challenge.
func ChallengeByName(db DB, name string) (*Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, designer, flag, total, body, hidden, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE name = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, name).Scan(&c.ID, &c.Name, &c.Category, &c.Designer, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChallengeByID retrieves a row from 'cyboard.challenge' as a Challenge.
func ChallengeByID(db DB, id int) (*Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, designer, flag, total, body, hidden, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE id = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, id).Scan(&c.ID, &c.Name, &c.Category, &c.Designer, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// AllChallenges fetches all ctf challenges from the database,
// to be displayed to staff.
func AllChallenges(db DB) ([]Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, designer, flag, total, body, hidden, created_at, modified_at ` +
		`FROM challenge`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []Challenge{}
	for rows.Next() {
		x := Challenge{}
		err = rows.Scan(&x.ID, &x.Name, &x.Category, &x.Designer, &x.Flag, &x.Total, &x.Body,
			&x.Hidden, &x.CreatedAt, &x.ModifiedAt)
		if err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}

// ChallengeSlice is an array of challenges, suitable to insert many of at once.
type ChallengeSlice []Challenge

// Insert many ctf challenges into the database at once.
func (cs ChallengeSlice) Insert(db TXer) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, c := range cs {
		err := c.Insert(tx)
		if err != nil {
			return errors.WithMessage(err,
				fmt.Sprintf("insert challenges (challenge=%q)", c.Name))
		}
	}
	return tx.Commit()
}

// func ChallengesInGroups(db DB, groups []string) ([]Challenge, error) {}

// ChallengeView is a safe-for-public-display subset of fields over a CTF challenge.
type ChallengeView struct {
	ID       int    `json:"id"`       // id
	Name     string `json:"name"`     // name
	Category string `json:"category"` // category
	Body     string `json:"body"`     // body
}

// AllPublicChallenges fetches all non-hidden ctf challenges from the database,
// to be displayed to constestants.
func AllPublicChallenges(db DB) ([]ChallengeView, error) {
	const sqlstr = `SELECT id, name, category, body FROM challenge WHERE hidden = false`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []ChallengeView{}
	for rows.Next() {
		x := ChallengeView{}
		if err = rows.Scan(&x.ID, &x.Name, &x.Category, &x.Body); err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}
