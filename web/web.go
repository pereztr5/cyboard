package web

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
)

func Start(addr string) {
	webRoutes := CreateWebRoutes()

	scoreEngine := NewScoreEngineAPI()
	apiRoutes := CreateAPIRoutes(scoreEngine)

	router := NewRouter(append(webRoutes, apiRoutes...))

	app := negroni.New()
	app.Use(negroni.NewLogger())
	app.Use(negroni.NewStatic(http.Dir("public")))

	app.UseHandler(router)

	l := log.New(os.Stdout, "[negroni] ", 0)
	l.Printf("listening on %s", addr)
	l.Fatal(http.ListenAndServeTLS(addr, "certs/cert.pem", "certs/key.pem", app))
}
