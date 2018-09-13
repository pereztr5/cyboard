package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/urfave/negroni"
)

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
	root.Post("/login", SubmitLogin)
	root.Get("/logout", Logout)
	root.Get("/scoreboard", ShowScoreboard)
	root.Get("/services", ShowServices)

	// Authenticated Pages for Blue Teams
	root.Group(func(authed chi.Router) {
		authed.Use(RequireLogin)
		authed.Get("/dashboard", ShowTeamDashboard)
		authed.Get("/challenges", ShowChallenges)
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
		blue.Use(RequireLogin)
		blue.Get("/challenges", GetPublicChallenges)
		blue.Post("/challenges", SubmitFlag)
	})

	// Staff API to view & edit the CTF event
	api.Route("/ctf", func(ctfStaff chi.Router) {
		ctfStaff.Use(RequireLogin, RequireCtfStaff)
		ctfStaff.Get("/stats/subs_per_flag", GetBreakdownOfSubmissionsPerFlag)
		ctfStaff.Get("/stats/teams_flags", GetEachTeamsCapturedFlags)

		ctfStaff.Route("/flags", func(r chi.Router) {
			r.Get("/", GetAllFlags)
			r.Post("/", AddFlags) // Insert many ctf challenges

			r.Get("/{id}", GetFlagByID)
			r.Put("/{id}", UpdateFlag)
			r.Delete("/{id}", DeleteFlag)
		})
	})

	// Admin API
	api.Route("/admin", func(admin chi.Router) {
		admin.Use(RequireLogin, RequireAdmin)

		admin.Post("/grant_bonus", GrantBonusPoints)

		admin.Route("/teams", func(r chi.Router) {
			r.Get("/", GetAllTeams)

			r.Put("/{id}", UpdateTeam)
			r.Delete("/{id}", DeleteTeam)
		})

		admin.Post("/blueteams", AddBlueteams) // Insert many blueteams

		admin.Route("/services", func(r chi.Router) {
			r.Get("/", GetAllServices)
			r.Post("/", AddService) // Insert one service

			r.Get("/{id}", GetServiceByID)
			r.Put("/{id}", UpdateService)
			r.Delete("/{id}", DeleteService)
		})
	})

	root.Mount("/api/", api)
	router.Mount("/", root)

	return router
}
