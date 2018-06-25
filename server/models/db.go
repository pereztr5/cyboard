// Package models contains the types for schema 'cyboard'.
package models

import (
	"github.com/jackc/pgx"
)

// DB is a type alias to `*pgx.ConnPool`, or some SQL database abstraction.
// Should the database connector change to something like libpq, sqlx, or dbr,
// the migration can be made easier.
type DB = *pgx.ConnPool

var (
	// DatabaseTables is a list of every table for the schema 'cyboard'
	DatabaseTables = []string{
		"challenge",
		"challenge_category",
		"challenge_file",
		"ctf_solve",
		"exit_status",
		"other_points",
		"service",
		"service_check",
		"team",
		"team_role",
	}
)
