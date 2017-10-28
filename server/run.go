package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/urfave/negroni"
)

type LogSettings struct {
	Level  string `mapstructure:"level"`
	Stdout bool   `mapstructure:"stdout"`
}

type DBSettings struct {
	URI    string `mapstructure:"mongodb_uri"`
	DBName string
}

type ServerSettings struct {
	IP                string
	HTTPPort          string   `mapstructure:"http_port"`
	HTTPSPort         string   `mapstructure:"https_port"`
	CertPath          string   `mapstructure:"cert"`
	CertKeyPath       string   `mapstructure:"key"`
	SpecialChallenges []string `mapstructure:"special_challenges"`
}

type Configuration struct {
	Appname  string      `mapstructure:"appname"`
	Log      LogSettings `mapstructure:"log"`
	Server   ServerSettings
	Database DBSettings
}

func Run(cfg *Configuration) {
	// Verify web app template files are available in working dir
	ensureAppTemplates()

	// Setup logs
	SetupScoringLoggers(&cfg.Log)
	Logger.Infof("%+v", cfg)
	// MongoDB setup
	SetupMongo(&cfg.Database, cfg.Server.SpecialChallenges)
	CreateIndexes()
	// Web Server Setup
	CreateStore()
	// On first run, prompt to set up an admin user
	EnsureAdmin()

	teamScoreUpdater, servicesUpdater := TeamScoreWsServer(), ServiceStatusWsServer()
	defer teamScoreUpdater.Stop()
	defer servicesUpdater.Stop()
	webRouter := CreateWebRouter(teamScoreUpdater, servicesUpdater)
	teamRouter, blackTeamRouter := CreateTeamRouter()
	adminRouter := CreateAdminRouter()

	app := negroni.New()
	basicAuth := negroni.New(negroni.HandlerFunc(RequireLogin))

	webRouter.PathPrefix("/admin").Handler(basicAuth.With(
		negroni.HandlerFunc(RequireAdmin),
		negroni.Wrap(adminRouter),
	))

	webRouter.PathPrefix("/black").Handler(basicAuth.With(
		negroni.HandlerFunc(RequireGroupIsAnyOf([]string{"admin", "blackteam"})),
		negroni.Wrap(blackTeamRouter),
	))

	webRouter.PathPrefix("/").Handler(basicAuth.With(
		negroni.Wrap(teamRouter),
	))

	app.Use(negroni.NewRecovery())
	app.Use(negroni.NewStatic(http.Dir("static")))
	app.Use(negroni.HandlerFunc(CheckSessionID))
	app.Use(RequestLogger)

	app.UseHandler(webRouter)

	sc := &cfg.Server

	if sc.CertPath == "" || sc.CertKeyPath == "" {
		Logger.Warn("SSL certs is not configured properly. Serving plain HTTP.")
		Logger.Printf("Server running at: http://%s:%s", sc.IP, sc.HTTPPort)
		Logger.Fatal(http.ListenAndServe(":"+sc.HTTPPort, app))
	} else {
		Logger.Printf("Server running at: http://%s:%s", sc.IP, sc.HTTPPort)
		Logger.Printf("Server running at: https://%s:%s", sc.IP, sc.HTTPSPort)
		go http.ListenAndServe(":"+sc.HTTPPort, http.HandlerFunc(redirecter(sc.HTTPSPort)))
		Logger.Fatal(http.ListenAndServeTLS(":"+sc.HTTPSPort, sc.CertPath, sc.CertKeyPath, app))
	}
}

func redirecter(port string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(fmt.Sprintf("http://%s", r.Host))
		if err != nil {
			Logger.Println("Error redirecting:", err)
			errCode := http.StatusInternalServerError
			http.Error(w, http.StatusText(errCode), errCode)
		}

		dest := fmt.Sprintf("https://%s:%s%s", u.Hostname(), port, r.URL.Path)
		http.Redirect(w, r, dest, http.StatusMovedPermanently)
	}
}
