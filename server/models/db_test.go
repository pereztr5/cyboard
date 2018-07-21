package models

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/sirupsen/logrus"
	testfixtures "gopkg.in/testfixtures.v2"
)

const connString = "host=localhost port=5432 dbname=cyboard_test user=cybot connect_timeout=10 sslmode=disable"

var (
	logger *logrus.Logger

	db       DBClient
	rawDB    *pgx.ConnPool
	stdlibDB *sql.DB // golang standard lib compatible connection for 'database/sql'
	fixtures *testfixtures.Context
)

func checkErr(err error, context string) {
	if err != nil {
		logger.WithError(err).Fatal(context)
	}
}

func TestMain(m *testing.M) {
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	setupDB()
	var err error
	fixtures, err = testfixtures.NewFolder(stdlibDB, &testfixtures.PostgreSQL{}, "testdata/fixtures")
	checkErr(err, "generating fixtures")
	m.Run()
}

func setupDB() {
	cfg, err := pgx.ParseDSN(connString)
	checkErr(err, "ParseDSN")
	cfg.Logger = logrusadapter.NewLogger(logger)

	rawDB, err = pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: cfg})
	checkErr(err, "create PG conn pool")
	db = rawDB

	stdlibDB, err = sql.Open("pgx", connString)
	checkErr(err, "create PG conn compatible with stdlib sql type")
}

func prepareTestDatabase(t *testing.T) {
	if err := fixtures.Load(); err != nil {
		t.Log(err)
		t.Fail()
	}
}
