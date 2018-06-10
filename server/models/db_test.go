package models

import (
	"testing"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/sirupsen/logrus"
)

var (
	testDB DB
	logger *logrus.Logger
)

func TestMain(m *testing.M) {
	setupDB()
	m.Run()
}

func setupDB() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	connPoolConfig := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     "127.0.0.1",
			User:     "cybot",
			Password: "",
			Database: "cyboard_test",
			Logger:   logrusadapter.NewLogger(logger),
		},
	}
	pool, err := pgx.NewConnPool(connPoolConfig)
	if err != nil {
		logger.Fatal("Unable to create connection pool", "error", err)
	}

	testDB = pool
}
