package apptest

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	_ "github.com/jackc/pgx/stdlib" // Sql driver
	"github.com/sirupsen/logrus"
	testfixtures "github.com/tbutts/testfixtures"
)

// connString enables connecting to Postgres as a regular user. Used to init `DB`, which
// is just like the connection would be in production, no mocks or anything like that.
const connString = "host=localhost port=5432 dbname=cyboard_test user=cybot connect_timeout=10 sslmode=disable"

// connStringSU enables connecting to Postgres as a super user. This is required for `StdlibDB`,
// which is used by the testfixtures library to reset the DB inbetween tests, and to do
// that it requires super user privs.
const connStringSU = "host=localhost port=5432 dbname=cyboard_test user=supercybot connect_timeout=10 sslmode=disable"

// If the CYTEST_LOG_SQL variable is set, all SQL queries will be logged. Can be useful in debugging.
// Another, more thorough option is to go into your postgresql.conf in your pg data dir,
// and set `log_statement = 'mod'`, which causes db/data altering statements to get logged.
const cytestLogSQL = "CYTEST_LOG_SQL"

// testFixtureFiles retrieves the paths to test data files.
// When adding more test data, this function will need to be updated.
func testFixtureFiles(testdataPath string) []string {
	// The order of the files in the array is the order they will be loaded into
	// the database before each test.
	// Be careful changing this! The testfixtures library may swallow INSERT stmt errors.
	files := []string{"team", "challenge", "challenge_file", "ctf_solve", "service", "service_check", "other_points"}
	for i, filename := range files {
		files[i] = fmt.Sprintf("%s/%s.yml", testdataPath, filename)
	}
	return files
}

var (
	DB       *pgx.ConnPool // db connection used throughout testing
	StdlibDB *sql.DB       // golang standard lib compatible db for 'database/sql'

	fixtures *testfixtures.Context
	logger   *logrus.Logger
)

func checkErr(err error, context string) {
	if err != nil {
		logger.WithError(err).Fatal(context)
	}
}

// Setup should be called in TestMain, to assign the necessary globals used
// behind the scenes in all database-related tests (that is, db conns, loggers,
// and test data). After this is run, tests should call PrepDatabase() before
// doing work, to ensure the database state matches the test data.
func Setup(testdataPath string) {
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	setupDB()

	var err error
	files := testFixtureFiles(testdataPath)
	fixtures, err = testfixtures.NewFiles(StdlibDB, &testfixtures.PostgreSQL{UseAlterConstraint: true}, files...)
	checkErr(err, "setting fixtures")
}

func setupDB() {
	cfg, err := pgx.ParseDSN(connString)
	checkErr(err, "ParseDSN")
	if os.Getenv(cytestLogSQL) != "" {
		cfg.Logger = logrusadapter.NewLogger(logger)
	}

	DB, err = pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: cfg})
	checkErr(err, "create PG conn pool")

	StdlibDB, err = sql.Open("pgx", connStringSU)
	checkErr(err, "create PG conn compatible with stdlib sql type")
}

// PrepDatabase readies the db into a clean state. This should be called before
// doing any interaction with the database in each test.
func PrepDatabase(t testing.TB) {
	if err := fixtures.Load(); err != nil {
		t.Fatalf("error preparing test database: %+v", err)
	}
}
