package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"golang.org/x/crypto/bcrypt"
)

var appCfg Configuration

func Run(cfg *Configuration) {
	appCfg = *cfg

	// Verify web app template files are available in working dir
	ensureAppTemplates()

	// Setup logs
	SetupScoringLoggers(&cfg.Log)
	Logger.Infof("%+v", cfg)

	setupResponder(Logger)

	// Postgres setup
	SetupPostgres(cfg.Database.URI)

	// Web Server Setup
	isHTTPS := cfg.Server.CertPath != "" && cfg.Server.CertKeyPath != ""
	CreateStore(isHTTPS)
	// On first run, prompt to set up an admin user
	EnsureAdmin(db)

	teamScoreUpdater, servicesUpdater := TeamScoreWsServer(), ServiceStatusWsServer()
	defer teamScoreUpdater.Stop()
	defer servicesUpdater.Stop()
	app := CreateWebRouter(teamScoreUpdater, servicesUpdater)

	sc := &cfg.Server

	if !isHTTPS {
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
			return
		}

		dest := fmt.Sprintf("https://%s:%s%s", u.Hostname(), port, r.URL.Path)
		http.Redirect(w, r, dest, http.StatusMovedPermanently)
	}
}

// EnsureAdmin helps bootstrap the app configuration by prompting & setting up
// an admin account if there is not one already configured.
func EnsureAdmin(db models.DB) {
	const sqlstr = `SELECT id FROM cyboard.team WHERE role_name = 'admin' LIMIT 1`
	err := db.QueryRow(sqlstr).Scan(nil)

	if err == nil {
		return
	} else if err != pgx.ErrNoRows {
		Logger.WithError(err).Fatal("EnsureAdmin: failed to check for team with admin privs.")
	}

	// No admin account enabled.
	// Read initial password from command line
	const adminAccName = "admin"
	fmt.Printf("*** No previously configured admin user found ***\n"+
		"Setting up %q account.\n"+
		"Provide a password for the account (you can change it later on the website):\n",
		adminAccName)
	fmt.Print(">> ")
	pass, err := ReadStdinLine()
	if err != nil {
		Logger.WithError(err).Fatal("EnsureAdmin: failed to read creds from stdin.")
	}
	hashBytes, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
	if err != nil {
		Logger.WithError(err).Fatal("EnsureAdmin: failed to hash password.")
	}

	newAdmin := &models.Team{
		Name:     adminAccName,
		RoleName: models.TeamRoleAdmin,
		Hash:     hashBytes,
	}
	err = newAdmin.Insert(db)
	if err != nil {
		Logger.WithError(err).Fatal("EnsureAdmin: failed to save admin account.")
	}

	fmt.Printf("%q account configured.\n", adminAccName)
	fmt.Println("Log in on the website to finish other configurations.")
}
