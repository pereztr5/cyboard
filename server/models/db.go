// Package models contains the types for schema 'cyboard'.
package models

import (
	"github.com/jackc/pgx"
)

// DB is the common interface for database operations that can be used with
// types from schema 'cyboard'.
type DB interface {
	Exec(string, ...interface{}) (pgx.CommandTag, error)
	Query(string, ...interface{}) (*pgx.Rows, error)
	QueryRow(string, ...interface{}) *pgx.Row
}

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
