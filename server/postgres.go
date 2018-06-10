package server

import (
	"github.com/jackc/pgx"
)

var (
	db *pgx.ConnPool
)

func SetupPostgres(uri string) {
	if ur == "" {
		Logger.Fatal("No postgres-uri specified.")
	}

	baseCfg, err := pgx.ParseConnectionString(uri)
	if err != nil {
		Logger.Errorf("SetupPostgres: uri parsing failed: uri=`%v`, error=%v", uri, err)
		Logger.Fatal("Did you check the docs? https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING")
	}

	// db is a globally available Postgres session generator
	db, err = pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: baseCfg})
	if err != nil {
		cfgStr := PgConfigAsString(&baseCfg)
		Logger.WithError(err).
			WithField("uhoh", "pool's closed, fool").WithField("config", cfgStr).
			Fatal("SetupPostgres: failed to create connection pool")
	}

	Logger.Info("Connected to postgres: ", PgConfigAsString(&baseCfg))
}
