package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/sirupsen/logrus"
)

var (
	rawDB *pgx.ConnPool
	db    models.DBClient
)

func SetGlobalPostgresDBs(pool *pgx.ConnPool) {
	rawDB, db = pool, pool
}

func SetupPostgres(uri string) *pgx.ConnPool {
	if rawDB != nil {
		// Database connection is already set up
		return rawDB
	}
	if uri == "" {
		Logger.Fatal("No postgres-uri specified.")
	}

	baseCfg, err := pgx.ParseConnectionString(uri)
	if err != nil {
		Logger.Errorf("SetupPostgres: uri parsing failed: uri=`%v`, error=%v", uri, err)
		Logger.Fatal("Did you check the docs? https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING")
	}

	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: baseCfg})
	if err != nil {
		cfgStr := PgConfigAsString(&baseCfg)
		Logger.WithFields(logrus.Fields{
			"error":  err,
			"config": cfgStr,
			"uhoh":   "pool's closed, fool",
		}).Fatal("SetupPostgres: failed to create connection pool")
	}
	SetGlobalPostgresDBs(pool)

	Logger.Info("Connected to postgres: ", PgConfigAsString(&baseCfg))
	return pool
}

func PingDB(ctx context.Context) error {
	if rawDB == nil {
		return errors.New("db is nil (no connection)")
	}
	conn, err := rawDB.Acquire()
	defer rawDB.Release(conn)
	if err != nil {
		return err
	}
	return conn.Ping(ctx)
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
