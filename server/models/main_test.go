package models

import (
	"testing"

	"github.com/pereztr5/cyboard/server/apptest"
)

var db DBClient

var prepareTestDatabase = apptest.PrepDatabase

func TestMain(m *testing.M) {
	apptest.Setup("../models/testdata/fixtures")
	db = apptest.DB
	m.Run()
}
