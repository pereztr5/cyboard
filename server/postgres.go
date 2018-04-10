package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

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
	pgStmtScoreCapturedFlag = "scoreCapturedFlag"
)

var (
	pgPool *pgx.ConnPool

	pgResultTable    = pgx.Identifier{"cy", "result"}
	pgServiceColumns = []string{
		"timestamp", "teamname", "teamnumber", "points", "type", "category", "exit_status",
	}
	pgCtfColumns = []string{
		"timestamp", "teamname", "teamnumber", "points", "type", "category", "challenge_name",
	}
)

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

func PostgresScoreCapturedFlag(res *Result) error {
	if pgPool == nil {
		return nil
	}

	_, err := pgPool.Exec(pgStmtScoreCapturedFlag,
		res.Timestamp, res.Teamname, int16(res.Teamnumber), int32(res.Points), res.Type, res.Group, res.Details)
	return err
}

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

func PostgresScoreServices(results []Result) error {
	return PostgresScoreMany(results, Service)
}

func PostgresScoreBonusFlags(results []Result) error {
	return PostgresScoreMany(results, CTF)
}

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
