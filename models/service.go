// Package models contains the types for schema 'cyboard'.
package models

import (
	"time"
)

// Service represents a row from 'cyboard.service'.
type Service struct {
	ID          int      `json:"id"`           // id
	Name        string   `json:"name"`         // name
	Category    string   `json:"category"`     // category
	Description string   `json:"description"`  // description
	TotalPoints float32  `json:"total_points"` // total_points
	Points      *float32 `json:"points"`       // points
	Script      string   `json:"script"`       // script
	Args        []string `json:"args"`         // args
	Disabled    bool     `json:"disabled"`     // disabled

	StartsAt   time.Time `json:"starts_at"`   // starts_at
	CreatedAt  time.Time `json:"created_at"`  // created_at
	ModifiedAt time.Time `json:"modified_at"` // modified_at

}

// Insert inserts the Service to the database.
func (s *Service) Insert(db DB) error {
	const sqlstr = `INSERT INTO cyboard.service (` +
		`name, category, description, total_points, points, script, args, disabled, starts_at, created_at, modified_at` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11` +
		`) RETURNING id`

	return db.QueryRow(sqlstr, s.Name, s.Category, s.Description, s.TotalPoints, s.Points, s.Script, s.Args, s.Disabled, s.StartsAt, s.CreatedAt, s.ModifiedAt).Scan(&s.ID)
}

// Update updates the Service in the database.
func (s *Service) Update(db DB) error {
	const sqlstr = `UPDATE cyboard.service SET (` +
		`name, category, description, total_points, points, script, args, disabled, starts_at` +
		`) = ( ` +
		`$2, $3, $4, $5, $6, $7, $8, $9, $10` +
		`) WHERE id = $1`
	_, err := db.Exec(sqlstr, s.ID, s.Name, s.Category, s.Description, s.TotalPoints, s.Points, s.Script, s.Args, s.Disabled, s.StartsAt)
	return err
}

// Delete deletes the Service from the database.
func (s *Service) Delete(db DB) error {
	const sqlstr = `DELETE FROM cyboard.service WHERE id = $1`
	_, err := db.Exec(sqlstr, s.ID)
	return err
}

// ServiceByName retrieves a row from 'cyboard.service' as a Service.
func ServiceByName(db DB, name string) (*Service, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, description, total_points, points, script, args, disabled, starts_at, created_at, modified_at ` +
		`FROM cyboard.service ` +
		`WHERE name = $1`
	s := Service{}
	err := db.QueryRow(sqlstr, name).Scan(&s.ID, &s.Name, &s.Category, &s.Description, &s.TotalPoints, &s.Points, &s.Script, &s.Args, &s.Disabled, &s.StartsAt, &s.CreatedAt, &s.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// ServiceByID retrieves a row from 'cyboard.service' as a Service.
func ServiceByID(db DB, id int) (*Service, error) {
	const sqlstr = `SELECT ` +
		`id, name, category, description, total_points, points, script, args, disabled, starts_at, created_at, modified_at ` +
		`FROM cyboard.service ` +
		`WHERE id = $1`
	s := Service{}
	err := db.QueryRow(sqlstr, id).Scan(&s.ID, &s.Name, &s.Category, &s.Description, &s.TotalPoints, &s.Points, &s.Script, &s.Args, &s.Disabled, &s.StartsAt, &s.CreatedAt, &s.ModifiedAt)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
