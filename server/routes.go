package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/urfave/negroni"
)

type methodHandlers map[string]http.HandlerFunc

func handleMethods(router *mux.Router, path string, methodHandlers methodHandlers) *mux.Router {
	rtr := router.Path(path).Subrouter()
	for method, fn := range methodHandlers {
		rtr.Methods(method).HandlerFunc(fn)
	}
	return rtr
}

func CreateWebRouter(teamScoreUpdater, servicesUpdater *broadcastHub) *mux.Router {
	router := mux.NewRouter()
	// Split off static asset handler, so that none of the other standard middleware gets run for static assets.
	router.PathPrefix("/assets/").Handler(http.FileServer(http.Dir("./static")))

	// Base router for server-side rendered content & API routes, with common middleware stack
	root := router.PathPrefix("/").Subrouter()
	root.Use(
		NegroniResponseWriterMiddleware,
		UnwrapNegroniMiddleware(negroni.NewRecovery()),
		sessionManager.Use,
		CheckSessionID,
		UnwrapNegroniMiddleware(RequestLogger),
	)

	// Public Template Pages
	root.HandleFunc("/", ShowHome).Methods("GET")
	root.HandleFunc("/login", ShowLogin).Methods("GET")
	root.HandleFunc("/login", SubmitLogin).Methods("POST")
	root.HandleFunc("/logout", Logout).Methods("GET")
	root.HandleFunc("/scoreboard", ShowScoreboard).Methods("GET")
	root.HandleFunc("/services", ShowServices).Methods("GET")

	// Authenticated Pages for Blue Teams
	authed := alice.New(RequireLogin)
	root.Handle("/dashboard", authed.ThenFunc(ShowTeamDashboard)).Methods("GET")
	root.Handle("/challenges", authed.ThenFunc(ShowChallenges)).Methods("GET")

	api := root.PathPrefix("/api/").Subrouter()

	// Public API
	public := api.PathPrefix("/public/").Subrouter()
	public.HandleFunc("/scores", GetScores).Methods("GET")
	public.HandleFunc("/scores/split", GetScoresSplit).Methods("GET")
	public.HandleFunc("/services", GetServices).Methods("GET")
	public.Handle("/scores/live", teamScoreUpdater.ServeWs()).Methods("GET")
	public.Handle("/services/live", servicesUpdater.ServeWs()).Methods("GET")

	// Blue Team API
	blue := api.PathPrefix("/blue/").Subrouter()
	blue.Use(RequireLogin)
	blue.HandleFunc("/challenges", GetPublicChallenges).Methods("GET")
	blue.HandleFunc("/challenges", SubmitFlag).Methods("POST")

	// Black Team API
	black := api.PathPrefix("/black/").Subrouter()
	black.Use(
		RequireLogin,
		RequireGroupIsAnyOf([]string{"admin", "blackteam"}),
	)
	black.HandleFunc("/grant_bonus", GrantBonusPoints).Methods("POST")

	// Staff API to view & edit the CTF event
	ctfStaff := api.PathPrefix("/ctf/").Subrouter()
	ctfStaff.Use(RequireLogin, RequireCtfGroupOwner)
	ctfStaff.HandleFunc("/stats/subs_per_flag", GetBreakdownOfSubmissionsPerFlag).Methods("GET")
	ctfStaff.HandleFunc("/stats/teams_flags", GetEachTeamsCapturedFlags).Methods("GET")

	handleMethods(ctfStaff, "/flags", methodHandlers{
		"GET":  GetConfigurableFlags,
		"POST": AddFlags,
	})
	handleMethods(ctfStaff, "/flags/{flag}", methodHandlers{
		"GET":    GetFlagByName,
		"POST":   AddFlag,
		"PUT":    UpdateFlag,
		"DELETE": DeleteFlag,
	})

	// Admin API
	admin := api.PathPrefix("/admin/").Subrouter()
	admin.Use(RequireLogin, RequireAdmin)
	handleMethods(admin, "/teams", methodHandlers{
		"GET":  GetAllUsers,
		"POST": AddTeams,
	})
	handleMethods(admin, "/team/{teamName}", methodHandlers{
		"PUT":    UpdateTeam,
		"DELETE": DeleteTeam,
	})

	return router
}
