package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/viper"
	"github.com/urfave/negroni"
)

var (
	specialChallenges []string
)

func SetupServerCfg(cfg *viper.Viper) {
	specialChallenges = cfg.GetStringSlice("server.special_challenges")
}

func Run() {
	// Setup logs
	SetupScoringLoggers(viper.GetViper())
	// MongoDB setup
	CreateIndexes()
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

	Logger.Printf("Server running at: http://%s:%s", viper.GetString("server.ip"), http_port)
	Logger.Printf("Server running at: https://%s:%s", viper.GetString("server.ip"), https_port)

	go http.ListenAndServe(":"+http_port, http.HandlerFunc(redir))

	Logger.Fatal(http.ListenAndServeTLS(":"+https_port, cert, key, app))
}

func redir(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse("http://" + r.Host)
	if err != nil {
		Logger.Printf("Error redirecting: %s\n", err)
	}

	http.Redirect(w, r, fmt.Sprintf("https://%s:%s%s",
		u.Hostname(), viper.GetString("server.https_port"), r.URL.Path), http.StatusMovedPermanently)
}
