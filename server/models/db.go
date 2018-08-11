// Package models contains the types for schema 'cyboard'.
package models

import (
	"context"

	"github.com/jackc/pgx"
)

// DBClient is a postgres session/engine/client. This is the main interface
// for doing database related stuff. It can be used to query, insert,
// and start transactions.
type DBClient interface {
	DB
	TXer
}

// DB is the common interface for database operations that can be used with
// types from schema 'cyboard'.
type DB interface {
	Exec(string, ...interface{}) (pgx.CommandTag, error)
	Query(string, ...interface{}) (*pgx.Rows, error)
	QueryRow(string, ...interface{}) *pgx.Row
	CopyFrom(tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int, error)
}

// TXer can start an isolated transaction in the database.
type TXer interface {
	Begin() (Tx, error)
	BeginEx(context.Context, *pgx.TxOptions) (Tx, error)
}

// Tx is a database connection that is in the middle of a transaction.
// Transactions can rollback all operations done during their lifetime,
// which helps maintain the database state.
type Tx = *pgx.Tx

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
