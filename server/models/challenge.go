// Package models contains the types for schema 'cyboard'.
package models

import (
	"time"
)

// Challenge represents a row from 'cyboard.challenge'.
type Challenge struct {
	ID               int     `json:"id"`                // id
	Name             string  `json:"name"`              // name
	Category         string  `json:"category"`          // category
	Flag             string  `json:"flag"`              // flag
	Total            float32 `json:"total"`             // total
	Body             string  `json:"body"`              // body
	Hidden           bool    `json:"hidden"`            // hidden
	DesignerCategory string  `json:"designer_category"` // designer_category

	CreatedAt  time.Time `json:"created_at"`  // created_at
	ModifiedAt time.Time `json:"modified_at"` // modified_at
}

// Insert inserts the Challenge to the database.
func (c *Challenge) Insert(db DB) error {
	const sqlstr = `INSERT INTO cyboard.challenge (` +
		`name, category, flag, total, body, hidden, designer_category` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7` +
		`) RETURNING id`

	return db.QueryRow(sqlstr, c.Name, c.Category, c.Flag, c.Total, c.Body, c.Hidden, c.DesignerCategory).Scan(&c.ID)
}

// Update updates the Challenge in the database.
func (c *Challenge) Update(db DB) error {
	const sqlstr = `UPDATE cyboard.challenge SET (` +
		`name, category, flag, total, body, hidden, designer_category` +
		`) = ( ` +
		`$2, $3, $4, $5, $6, $7, $8` +
		`) WHERE id = $1`

	_, err := db.Exec(sqlstr, c.ID, c.Name, c.Category, c.Flag, c.Total, c.Body, c.Hidden, c.DesignerCategory)
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
		`id, name, category, flag, total, body, hidden, designer_category, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE flag = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, flag).Scan(&c.ID, &c.Name, &c.Category, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.DesignerCategory, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChallengeByName retrieves a row from 'cyboard.challenge' as a Challenge.
func ChallengeByName(db DB, name string) (*Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, flag, total, body, hidden, designer_category, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE name = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, name).Scan(&c.ID, &c.Name, &c.Category, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.DesignerCategory, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ChallengeByID retrieves a row from 'cyboard.challenge' as a Challenge.
func ChallengeByID(db DB, id int) (*Challenge, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, flag, total, body, hidden, designer_category, created_at, modified_at ` +
		`FROM cyboard.challenge ` +
		`WHERE id = $1`

	c := Challenge{}
	err := db.QueryRow(sqlstr, id).Scan(&c.ID, &c.Name, &c.Category, &c.Flag, &c.Total, &c.Body, &c.Hidden, &c.DesignerCategory, &c.CreatedAt, &c.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
