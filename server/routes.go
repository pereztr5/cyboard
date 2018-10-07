package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/urfave/negroni"
)

// MaxReqsPerSec caps the requests per IP for rate-limited endpoints
// (if limiting is enabled w/ the config `service_monitor.rate_limit = "true"`).
//
// Currently, rate limiting is fixed to just a few endpoints where
// a blueteam can hit that endpoint at most once per second, full-stop.
const MaxReqsPerSec = 1

func CreateWebRouter(teamScoreUpdater, servicesUpdater *broadcastHub) chi.Router {
	router := chi.NewRouter()
	// Split off static asset handler, so that none of the other standard middleware gets run for static assets.
	router.Handle("/assets/*", http.FileServer(http.Dir("./static")))

	// Health check
	router.HandleFunc("/ping", PingHandler)

	// Base router for server-side rendered content & API routes, with common middleware stack
	root := chi.NewRouter()
	root.Use(
		NegroniResponseWriterMiddleware,
		UnwrapNegroniMiddleware(negroni.NewRecovery()),
		sessionManager.Use,
		CheckSessionID,
		UnwrapNegroniMiddleware(RequestLogger),
	)

	// Public Template Pages
	root.Get("/", ShowHome)
	root.Get("/login", ShowLogin)
	MaybeRateLimit(root, MaxReqsPerSec).Post("/login", SubmitLogin)
	root.Get("/logout", Logout)
	root.Get("/scoreboard", ShowScoreboard)
	root.Get("/services", ShowServices)

	// Authenticated Pages for Blue Teams
	root.Group(func(authed chi.Router) {
		authed.Use(RequireLogin, RequireEventStarted)
		authed.Get("/dashboard", ShowTeamDashboard)
		authed.Get("/challenges", ShowChallenges)
	})

	// Pages for admins (configuration, analytic dashboards)
	root.Route("/admin", func(admin chi.Router) {
		admin.Use(RequireLogin, RequireAdmin)
		admin.Get("/bonuses", ShowBonusPage)
		admin.Get("/teams", ShowTeamsConfig)
	})

	// Pages for ctf creators
	root.Route("/staff", func(staff chi.Router) {
		staff.Use(RequireLogin, RequireCtfStaff)
		staff.Get("/ctf", ShowCtfConfig)
		staff.Get("/ctf_dashboard", ShowCtfDashboard)
	})

	api := chi.NewRouter()

	// Public API
	api.Route("/public", func(public chi.Router) {
		public.Get("/scores", GetScores)
		public.Get("/services", GetServicesStatuses)
		public.Handle("/scores/live", teamScoreUpdater.ServeWs())
		public.Handle("/services/live", servicesUpdater.ServeWs())
	})

	// Blue Team API
	api.Route("/blue", func(blue chi.Router) {
		blue.Use(RequireLogin, RequireEventStarted)
		blue.Get("/challenges", GetPublicChallenges)
		MaybeRateLimit(blue, MaxReqsPerSec).With(RequireEventNotOver).
			Post("/challenges", SubmitFlag)

		blue.Route("/challenges/{id}", func(r chi.Router) {
			r.Use(RequireIdParam)
			r.Get("/", GetChallengeDescription)
			r.Get("/files", CtfFileMgr.GetFileList)
			r.Get("/files/{name}", CtfFileMgr.GetFile)
		})
	})

	// Staff API to view & edit the CTF event
	api.Route("/ctf", func(ctfStaff chi.Router) {
		ctfStaff.Use(RequireLogin, RequireCtfStaff)
		ctfStaff.Get("/stats/subs_per_flag", GetBreakdownOfSubmissionsPerFlag)
		ctfStaff.Get("/stats/teams_flags", GetEachTeamsCapturedFlags)

		ctfStaff.Route("/flags", func(r chi.Router) {
			r.Get("/", GetAllFlags)
			r.Post("/", AddFlags) // Insert many ctf challenges

			r.Route("/{id}", func(r chi.Router) {
				r.Use(RequireIdParam)
				r.Get("/", GetFlagByID)
				r.Put("/", UpdateFlag)
				r.Delete("/", DeleteFlag)

				// `<host>/api/ctf/flags/4/files/suspicious.pdf`
				r.Route("/files", func(r chi.Router) {
					r.Get("/", CtfFileMgr.GetFileList)
					r.Post("/", CtfFileMgr.SaveFile)

					r.Get("/{name}", CtfFileMgr.GetFile)
					r.Delete("/{name}", CtfFileMgr.DeleteFile)
				})
			})
		})
	})

	// Admin API
	api.Route("/admin", func(admin chi.Router) {
		admin.Use(RequireLogin, RequireAdmin)

		admin.Post("/grant_bonus", GrantBonusPoints)

		admin.Route("/teams", func(r chi.Router) {
			r.Get("/", GetAllTeams)
			r.Post("/", AddTeam)

			r.Route("/{id}", func(r chi.Router) {
				r.Use(RequireIdParam)
				r.Get("/", GetTeamByID)
				r.Put("/", UpdateTeam)
				r.Delete("/", DeleteTeam)
			})
		})

		admin.Post("/blueteams", AddBlueteams) // Insert many blueteams

		admin.Route("/services", func(r chi.Router) {
			r.Get("/", GetAllServices)
			r.Post("/", AddService) // Insert one service

			r.Route("/{id}", func(r chi.Router) {
				r.Use(RequireIdParam)
				r.Get("/", GetServiceByID)
				r.Put("/", UpdateService)
				r.Delete("/", DeleteService)
			})
		})

		admin.Route("/scripts", func(r chi.Router) {
			r.Get("/", ScriptMgr.GetFileList)
			r.Post("/", ScriptMgr.SaveFile)

			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", ScriptMgr.GetFile)
				r.Delete("/", ScriptMgr.DeleteFile)

				r.Post("/run", RunScriptTest)
			})
		})
	})

	root.Mount("/api/", api)
	router.Mount("/", root)

	return router
}
