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
	router.With(Compress()).Handle("/assets/*", http.FileServer(http.Dir("./ui/static")))
	serveFile := http.FileServer(http.Dir("./ui/static/assets"))
	router.Handle("/robots.txt", serveFile)
	router.Handle("/favicon.ico", serveFile)

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
	pages := chi.NewRouter()
	pages.Use(Compress())

	pages.Get("/", ShowHome)
	pages.Get("/login", ShowLogin)
	MaybeRateLimit(pages, MaxReqsPerSec).Post("/login", SubmitLogin)
	pages.Get("/logout", Logout)

	pages.Group(func(r chi.Router) {
		r.Use(RequireEventStarted)
		r.Get("/scoreboard", ShowScoreboard)
		r.Get("/services", ShowServices)
	})

	// Authenticated Pages for Blue Teams
	pages.Group(func(authed chi.Router) {
		authed.Use(RequireLogin, RequireEventStarted)
		authed.Get("/dashboard", ShowTeamDashboard)
		authed.Get("/challenges", ShowChallenges)
	})

	// Pages for admins (configuration, analytic dashboards)
	pages.Route("/admin", func(admin chi.Router) {
		admin.Use(RequireLogin, RequireAdmin)
		admin.Get("/bonuses", ShowBonusPage)
		admin.Get("/teams", ShowTeamsConfig)
		admin.Get("/services", ShowServicesConfig)
		admin.Get("/services/scripts", ShowServiceScriptsConfig)
	})

	// Pages for ctf creators
	pages.Route("/staff", func(staff chi.Router) {
		staff.Use(RequireLogin, RequireCtfStaff)
		staff.Get("/ctf", ShowCtfConfig)
		staff.Get("/ctf_dashboard", ShowCtfDashboard)
		staff.Get("/log_files", ShowLogViewer)
	})

	api := chi.NewRouter()

	// Public API
	api.Route("/public", func(public chi.Router) {
		public.Get("/scores", GetScores)
		public.Get("/services", GetServicesStatuses)
		public.Handle("/scores/live", teamScoreUpdater.ServeWs())
		public.Handle("/services/live", servicesUpdater.ServeWs())

		public.Get("/ctf/solves", GetChallengeCapturesByTime)
	})

	// Blue Team API
	api.Route("/blue", func(blue chi.Router) {
		blue.Use(RequireLogin, RequireEventStarted)
		blue.Get("/challenges", GetPublicChallenges)
		MaybeRateLimit(blue, MaxReqsPerSec).With(RequireNotOnBreak(), RequireEventNotOver).
			Post("/challenges", SubmitFlag)

		blue.Route("/challenges/{id}", func(r chi.Router) {
			r.Use(RequireIdParam)
			r.Get("/", GetChallengeDescription)
			r.Get("/files", CtfFileMgr.GetFileList)
			r.Get("/files/{name}", CtfFileMgr.GetFile)
		})
	})

	// Staff API to view & edit the CTF event
	api.Route("/staff", func(staff chi.Router) {
		staff.Use(RequireLogin, RequireCtfStaff)

		staff.Get("/event_config", GetEventConfig)
	})

	// Staff API to view & edit the CTF event
	api.Route("/ctf", func(ctfStaff chi.Router) {
		ctfStaff.Use(RequireLogin, RequireCtfStaff)
		ctfStaff.Get("/stats/subs_per_flag", GetBreakdownOfSubmissionsPerFlag)
		ctfStaff.Get("/stats/teams_flags", GetEachTeamsCapturedFlags)

		ctfStaff.Route("/logs", func(r chi.Router) {
			r.Get("/", LogReadOnlyMgr.GetFileList)
			r.Get("/{name}", LogReadOnlyMgr.GetFile)
			r.Get("/{name}/tail", WsTailFile)
		})

		// TODO / HACK: Needed a place for a late-added "add one" challenge
		// This should be in the /flags namespace below, and the
		// insert many should be separate, or a query param toggle
		ctfStaff.Post("/new_flag", AddFlag)

		ctfStaff.Get("/flag", GetFlagByName)

		ctfStaff.Route("/flags", func(r chi.Router) {
			r.Get("/", GetAllFlags)
			r.Post("/", AddFlags) // Insert many ctf challenges

			r.Route("/{id}", func(r chi.Router) {
				r.Use(RequireIdParam)
				r.Get("/", GetFlagByID)
				r.Put("/", UpdateFlag)
				r.Delete("/", DeleteFlag)

				r.Post("/activate", EnableCTFChallenge)

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

		admin.Get("/all_bonus", GetBonusPoints)
		admin.Post("/grant_bonus", GrantBonusPoints)

		admin.Get("/team/{name}", GetTeamByName)

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

		admin.Route("/blueteams", func(r chi.Router) {
			r.Get("/", GetBlueteams)  // Get all non-disabled blueteams
			r.Post("/", AddBlueteams) // Insert many blueteams
		})

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
			//r.Post("/", ScriptMgr.SaveFile)

			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", ScriptMgr.GetFile)
				//r.Delete("/", ScriptMgr.DeleteFile)

				r.Post("/run", RunScriptTest)
			})
		})
	})

	root.Mount("/api/", api)
	root.Mount("/", pages)
	router.Mount("/", root)

	return router
}
