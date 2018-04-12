package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

// PostgresMetricsResult represents the `cy.result` table schema
// in the database.
//
// This table contains two similar but different `types`. The ChallengeName
// column is only used for "type = 'CTF'" rows, whereas the ExitStatus column
// is for "type = 'Service'" rows. There are better ways to implement this
// (such as JSONB columns or actually competent table design), but that can be
// fixed when there's more time.
//
// This type is not directly used at the moment, as most methods for insertion
// in `github.com/jackc/pgx` take explicit interface{} slices. There is no
// retrieval of rows from Postgres in cyboard, all those queries are done in
// Grafana, a data analysis tool. The type signature sits here for reference.
type PostgresMetricsResult struct {
	Timestamp     time.Time   `sql:"timestamp"`
	Teamname      string      `sql:"teamname"`
	Teamnumber    int16       `sql:"teamnumber"`
	Points        int32       `sql:"points"`
	Type          string      `sql:"type"`
	Category      string      `sql:"category"`
	ChallengeName pgtype.Text `sql:"challenge_name"`
	ExitStatus    pgtype.Int4 `sql:"exit_status"`
}

const (
	// pgStmtScoreCapturedFlag is the sole prepared statement, used to insert a scored flag into PG
	pgStmtScoreCapturedFlag = "scoreCapturedFlag"
)

var (
	// pgPool is the global conn for doing Postgres stuff.
	pgPool *pgx.ConnPool

	// pgResultTabe is the schema and table name for every teams' scores
	pgResultTable = pgx.Identifier{"cy", "result"}
	// pgServiceColumns are the names of the columns with values for "type = 'Service'" rows
	pgServiceColumns = []string{
		"timestamp", "teamname", "teamnumber", "points", "type", "category", "exit_status",
	}
	// pgServiceColumns are the names of the columns with values for "type = 'CTF'" rows
	pgCtfColumns = []string{
		"timestamp", "teamname", "teamnumber", "points", "type", "category", "challenge_name",
	}
)

// SetupPostgres connects to a PG instance using the given `uri` parameter.
// If the uri is empty, Postgres support is disabled. Otherwise, the global
// `pgPool` will be set up, which will signal the app to duplicate scores into
// Postgres as well as Mongodb.
func SetupPostgres(uri string) {
	// If the uri is unconfigured, skip saving data to Postgres
	if uri == "" {
		Logger.Warn("No postgres-uri specified. Disabling postgres support " +
			"(which means no metrics in Grafana!)")
		pgPool = nil
		return
	}

	// Passwords can be in the PostgresURI in the config, or in a .pgpass file
	baseCfg, err := pgx.ParseConnectionString(uri)
	if err != nil {
		Logger.Errorf("SetupPostgres: uri parsing failed: uri=`%v`, error=%v", uri, err)
		Logger.Fatal("Did you check the docs? https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING")
	}

	// pgPool is a globally available Postgres session generator
	pgPool, err = pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: baseCfg})
	if err != nil {
		// pool's closed, fool
		Logger.Fatalf("SetupPostgres: failed to create connection pool: config=%s, error=%v", PgConfigAsString(&baseCfg), err)
	}

	Logger.Info("Connected to postgres: ", PgConfigAsString(&baseCfg))

	// prepare statements
	if _, err := pgPool.Prepare(pgStmtScoreCapturedFlag,
		`INSERT INTO cy.result (timestamp, teamname, teamnumber, points, type, category, challenge_name)
		values ($1,$2,$3,$4,$5,$6,$7)
		`); err != nil {
		Logger.Fatalf("SetupPostgres: failed to prepare statement `%v`: %v", pgStmtScoreCapturedFlag, err)
	}
}

// PostgresScoreCapturedFlag saves a single scored flag into postgres `cy.result` table
func PostgresScoreCapturedFlag(res *Result) error {
	if pgPool == nil {
		return nil
	}

	_, err := pgPool.Exec(pgStmtScoreCapturedFlag,
		res.Timestamp, res.Teamname, int16(res.Teamnumber), int32(res.Points), res.Type, res.Group, res.Details)
	return err
}

// PostgresScoreMany saves a slice of results into Postgres.
// Either for 'CTF', or 'Service' type results.
func PostgresScoreMany(results []Result, category string) error {
	if pgPool == nil {
		return nil
	}

	var columns []string
	iresults := make([][]interface{}, len(results))
	if category == Service {
		columns = pgServiceColumns
		for i, r := range results {
			xs := strings.Split(r.Details, ": ")
			exitStatus, err := strconv.Atoi(xs[len(xs)-1])
			if err != nil {
				exitStatus = 3
			}
			iresults[i] = []interface{}{
				r.Timestamp, r.Teamname, int16(r.Teamnumber), int32(r.Points), r.Type, r.Group, exitStatus,
			}
		}
	} else {
		columns = pgCtfColumns
		for i, r := range results {
			iresults[i] = []interface{}{
				r.Timestamp, r.Teamname, int16(r.Teamnumber), int32(r.Points), r.Type, r.Group, r.Details,
			}
		}
	}

	_, err := pgPool.CopyFrom(
		pgResultTable,
		columns,
		pgx.CopyFromRows(iresults),
	)
	return err
}

// PostgresScoreServices saves service checker results into postgres
func PostgresScoreServices(results []Result) error {
	return PostgresScoreMany(results, Service)
}

// PostgresScoreServices saves bonus points/flags from results into postgres
func PostgresScoreBonusFlags(results []Result) error {
	return PostgresScoreMany(results, CTF)
}

// PgConfigAsString returns a string repr of a pgx.ConnConfig.
// The password field is bleeted out if it is not-empty.
func PgConfigAsString(c *pgx.ConnConfig) string {
	var pw string
	if c.Password != "" {
		pw = "*****"
	}

	port := c.Port
	if port == 0 {
		port = 5432
	}

	return fmt.Sprintf("pgx.ConnConfig{host=%q, port=%v, db=%q, user=%q, password=%q}",
		c.Host, port, c.Database, c.User, pw)
}
