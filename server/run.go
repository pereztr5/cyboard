package server

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
	"github.com/urfave/negroni"
)

func Run() {
	// Setup logs
	SetupScoringLoggers(viper.GetViper())
	// MongoDB setup
	CreateUniqueIndexes()
	// Web Server Setup
	CreateStore()
	// On first run, prompt to set up an admin user
	EnsureAdmin()

	webRouter := CreateWebRouter()
	teamRouter := CreateTeamRouter()
	adminRouter := CreateAdminRouter()

	app := negroni.New()
	basicAuth := negroni.New(negroni.HandlerFunc(RequireLogin))

	webRouter.PathPrefix("/admin").Handler(basicAuth.With(
		negroni.HandlerFunc(RequireAdmin),
		negroni.Wrap(adminRouter),
	))

	webRouter.PathPrefix("/").Handler(basicAuth.With(
		negroni.Wrap(teamRouter),
	))

	app.Use(negroni.NewRecovery())
	app.Use(negroni.NewStatic(http.Dir("static")))
	app.Use(negroni.HandlerFunc(CheckSessionID))
	app.Use(RequestLogger)

	app.UseHandler(webRouter)

	http_port := viper.GetString("server.http_port")
	https_port := viper.GetString("server.https_port")
	cert := viper.GetString("server.cert")
	key := viper.GetString("server.key")

	Logger.Printf("Server running at: https://%s:%s", viper.GetString("server.ip"), https_port)

	go http.ListenAndServe(":"+http_port, http.HandlerFunc(redir))

	Logger.Fatal(http.ListenAndServeTLS(":"+https_port, cert, key, app))
}

func redir(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("https://%s:%s/%s",
		viper.GetString("server.ip"), viper.GetString("https_port"), r.RequestURI), http.StatusMovedPermanently)
}
