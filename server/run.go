package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

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
	Logger.Debugf("%+v", cfg)

	setupResponder(Logger)

	// Postgres setup
	SetupPostgres(cfg.Database.URI)

	// Web Server Setup
	isHTTPS := cfg.Server.CertPath != "" && cfg.Server.CertKeyPath != ""
	CreateStore(isHTTPS)
	// On first run, prompt to set up an admin user
	EnsureAdmin(db)

	teamScoreUpdater, servicesUpdater := TeamScoreWsServer(), ServiceStatusWsServer()
	app := CreateWebRouter(teamScoreUpdater, servicesUpdater)

	// Setup http(s) server
	sc := &cfg.Server
	httpAddr := sc.IP + ":" + sc.HTTPPort
	httpsAddr := sc.IP + ":" + sc.HTTPSPort

	server := &http.Server{
		Handler:      app,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  90 * time.Second,
		TLSConfig:    tlsConfig(), // Ignored if only serving http
	}
	server.RegisterOnShutdown(teamScoreUpdater.Stop)
	server.RegisterOnShutdown(servicesUpdater.Stop)
	shutdownComplete := shutdownWatcher(server)

	var serveErr error

	if !isHTTPS {
		Logger.Warn("SSL certs is not configured properly. Serving plain HTTP.")
		Logger.Printf("Server running at: http://%s", httpAddr)
		server.Addr = httpAddr
		serveErr = server.ListenAndServe()
	} else {
		Logger.Printf("Server running at: http://%s | https://%s", httpAddr, httpsAddr)
		go http.ListenAndServe(httpAddr, http.HandlerFunc(redirecter(sc.HTTPSPort)))
		server.Addr = httpsAddr
		serveErr = server.ListenAndServeTLS(sc.CertPath, sc.CertKeyPath)
	}

	if serveErr != http.ErrServerClosed {
		Logger.WithError(serveErr).Error("Server crash!")
	}

	<-shutdownComplete
	Logger.Info("Server shutdown complete")
}

func redirecter(port string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		dest := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
		dest.Host = fmt.Sprintf("%s:%s", dest.Hostname(), port)
		http.Redirect(w, r, dest.String(), http.StatusTemporaryRedirect)
	}
}

// shutdownWatcher will stop the http server when a SIGINT is caught.
// Up to 5 seconds are given for connections to finish up.
func shutdownWatcher(srv *http.Server) chan struct{} {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		Logger.Info("Interrupt received - Server is shutting down")

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := srv.Shutdown(ctx); err != nil {
			Logger.WithError(err).Error("HTTP server Shutdown")
		}
		close(idleConnsClosed)
	}()

	return idleConnsClosed
}

// tlsConfig is the preferred, secure settings for web servers on the open web.
// (Date: October 2018)
func tlsConfig() *tls.Config {
	// Thanks to Filippo for a being a helpful fella on this one:
	// https://blog.cloudflare.com/exposing-go-on-the-internet/

	return &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// EnsureAdmin helps bootstrap the app configuration by prompting & setting up
// an admin account if there is not one already configured.
func EnsureAdmin(db models.DB) {
	const sqlstr = `SELECT id FROM team WHERE role_name = 'admin' LIMIT 1`
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
