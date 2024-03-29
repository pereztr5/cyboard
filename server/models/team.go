// Package models contains the types for schema 'cyboard'.
package models

import (
	"fmt"

	"github.com/pkg/errors"
)

// Team represents a row from 'cyboard.team'.
type Team struct {
	ID         int      `json:"id"`          // id
	Name       string   `json:"name"`        // name
	RoleName   TeamRole `json:"role_name"`   // role_name
	Hash       []byte   `json:"-"`           // hash
	Disabled   bool     `json:"disabled"`    // disabled
	BlueteamIP *int16   `json:"blueteam_ip"` // blueteam_ip
}

// Insert inserts the Team to the database.
func (t *Team) Insert(db DB) error {
	const sqlstr = `INSERT INTO team (` +
		`name, role_name, hash, disabled, blueteam_ip` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5` +
		`) RETURNING id`

	return db.QueryRow(sqlstr, t.Name, t.RoleName, t.Hash, t.Disabled, t.BlueteamIP).Scan(&t.ID)
}

// Update updates the Team in the database.
// If the `Hash` field is not set, then Update will not attempt to change
// the team's password.
func (t *Team) Update(db DB) error {
	var err error
	if t.Hash == nil {
		const sqlstr = `UPDATE team SET (name, role_name, disabled, blueteam_ip) = ($2, $3, $4, $5) WHERE id = $1`
		_, err = db.Exec(sqlstr, t.ID, t.Name, t.RoleName, t.Disabled, t.BlueteamIP)
	} else {
		const sqlstr = `UPDATE team SET (name, role_name, disabled, blueteam_ip, hash) = ($2, $3, $4, $5, $6) WHERE id = $1`
		_, err = db.Exec(sqlstr, t.ID, t.Name, t.RoleName, t.Disabled, t.BlueteamIP, t.Hash)
	}
	return err
}

// Delete deletes the Team from the database.
func (t *Team) Delete(db DB) error {
	const sqlstr = `DELETE FROM team WHERE id = $1`
	_, err := db.Exec(sqlstr, t.ID)
	return err
}

// TeamByName retrieves a row from 'cyboard.team' as a Team.
func TeamByName(db DB, name string) (*Team, error) {
	const sqlstr = `SELECT ` +
		`id, name, role_name, hash, disabled, blueteam_ip ` +
		`FROM team ` +
		`WHERE name = $1`
	t := Team{}
	err := db.QueryRow(sqlstr, name).Scan(&t.ID, &t.Name, &t.RoleName, &t.Hash, &t.Disabled, &t.BlueteamIP)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// TeamByID retrieves a row from 'cyboard.team' as a Team.
func TeamByID(db DB, id int) (*Team, error) {
	const sqlstr = `SELECT ` +
		`id, name, role_name, hash, disabled, blueteam_ip ` +
		`FROM team ` +
		`WHERE id = $1`
	t := Team{}
	err := db.QueryRow(sqlstr, id).Scan(&t.ID, &t.Name, &t.RoleName, &t.Hash, &t.Disabled, &t.BlueteamIP)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// AllTeams fetches all teams (users) from the database.
// Used by the admin dashboard to view & modify all added users.
func AllTeams(db DB) ([]Team, error) {
	const sqlstr = `SELECT id, name, role_name, disabled, blueteam_ip FROM team ` +
		`ORDER BY role_name DESC, id`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ts := []Team{}
	for rows.Next() {
		t := Team{}
		if err = rows.Scan(&t.ID, &t.Name, &t.RoleName, &t.Disabled, &t.BlueteamIP); err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("get all teams (team=%q)", t.Name))
		}
		ts = append(ts, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ts, nil
}

// BlueTeamStore contains the fields used to insert new blue teams into the database.
type BlueTeamStore struct {
	Name       string `json:"name"`        // name
	Hash       []byte `json:"-"`           // hash
	BlueteamIP int16  `json:"blueteam_ip"` // blueteam_ip
}

// BlueTeamStoreSlice is a list of blue teams, ready to be batch inserted.
type BlueTeamStoreSlice []BlueTeamStore

// Insert a batch of new blue teams into the database.
// Blue teams must have a unique name and ip from all other blueteams.
func (teams BlueTeamStoreSlice) Insert(db TXer) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const sqlstr = `INSERT INTO team (role_name, name, blueteam_ip, hash) VALUES ('blueteam', $1, $2, $3)`
	for _, t := range teams {
		_, err = tx.Exec(sqlstr, t.Name, t.BlueteamIP, t.Hash)
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("insert bluteams (team=%q)", t.Name))
		}
	}
	return tx.Commit()
}

// BlueteamView has the Team fields needed by the service monitor.
type BlueteamView struct {
	ID         int    `json:"id"`          // id
	Name       string `json:"name"`        // name
	BlueteamIP int16  `json:"blueteam_ip"` // blueteam_ip
}

// AllBlueteams fetches all non-disabled contestants from the database, along
// with their significant IP octet (the one octet that changes between teams,
// all the other octets are assumed to be the same).
func AllBlueteams(db DB) ([]BlueteamView, error) {
	const sqlstr = `SELECT ` +
		`id, name, blueteam_ip ` +
		`FROM team ` +
		`WHERE role_name = 'blueteam' AND disabled = false ` +
		`ORDER BY id`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bmvs := []BlueteamView{}
	for rows.Next() {
		b := BlueteamView{}
		if err = rows.Scan(&b.ID, &b.Name, &b.BlueteamIP); err != nil {
			return nil, err
		}
		bmvs = append(bmvs, b)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bmvs, nil
}
