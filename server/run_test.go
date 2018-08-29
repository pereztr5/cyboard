package server

import (
	"os"
	"testing"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/cookiestore"
	"github.com/pereztr5/cyboard/server/apptest"
)

func createTestLoginStore() {
	testSessionKey := []byte("WO-OOAH BLACK BETTY AMBERLAMPS!!")
	sessionManager = scs.NewManager(cookiestore.New(testSessionKey))
	sessionManager.Name("cyboard")
}

func TestMain(m *testing.M) {
	apptest.Setup("models/testdata/fixtures")

	SetupScoringLoggers(&LogSettings{Level: "debug", Stdout: true})
	setupResponder(Logger)
	SetGlobalPostgresDBs(apptest.DB)
	createTestLoginStore()

	os.Exit(m.Run())
}
