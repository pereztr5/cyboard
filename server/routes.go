package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/urfave/negroni"
)

type Page struct {
	Title string
	T     models.Team
}

var templates map[string]*template.Template

// Parse templates at startup
func ensureAppTemplates() {
	if templates != nil {
		return
	}

	templates = make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"title":              strings.Title,
		"totalChallenges":    getChallenges,
		"teamChallenges":     getTeamChallenges,
		"teamScore":          getTeamScore,
		"allTeamScores":      getAllTeamScores,
		"allBlueTeams":       DataGetTeams,
		"getOwnedChalGroups": getChallengesOwnerOf,
		"getStatus":          DataGetResultByService,
		"serviceList":        DataGetServiceList,
		"challengesList":     DataGetChallengeGroupsList,
		"allUsers":           DataGetAllUsers,
		"isChallengeOwner":   AllowedToConfigureChallenges,
		"existsSpecialFlags": existsSpecialFlags,
		"StringsJoin":        strings.Join,
	}

	includes := mustGlobFiles("tmpl/includes/*.tmpl")
	layouts := mustGlobFiles("tmpl/*.tmpl")

	for _, layout := range layouts {
		files := append(includes, layout)
		title := strings.TrimSuffix(filepath.Base(layout), ".tmpl")
		templates[title] = template.Must(template.New(layout).Funcs(funcMap).ParseFiles(files...))
	}
}

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
		UnwrapNegroniMiddleware(RequestLogger),
		CheckSessionID,
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
		RequireGroupIsAnyOf{[]string{"admin", "blackteam"}}.Middleware,
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

func ShowHome(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "homepage"}
	if t != nil {
		p.T = t.(models.Team)
	}
	renderTemplate(w, p)
}

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("team") == nil {
		p := Page{
			Title: "login",
		}
		renderTemplate(w, p)
	} else {
		http.Redirect(w, r, "/dashboard", 302)
	}
}

func SubmitLogin(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	//if err != nil {
	//	Logger.Warn("Getting session cookie from Store failed: ", err)
	//}

	succ := CheckCreds(w, r)
	if succ {
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.Redirect(w, r, "/dashboard", 302)
		return
	}
	http.Redirect(w, r, "/login", 302)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	delete(session.Values, "id")
	// Make sure we save the session after deleting the ID.
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	http.Redirect(w, r, "/login", 302)
}

func ShowTeamDashboard(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Title: "dashboard",
		T:     r.Context().Value("team").(models.Team),
	}
	renderTemplate(w, p)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	if t != nil {
		p := Page{
			Title: "challenges",
			T:     t.(models.Team),
		}
		renderTemplate(w, p)
	}
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "scoreboard"}
	if t != nil {
		p.T = t.(models.Team)
	}
	renderTemplate(w, p)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "services"}
	if t != nil {
		p.T = t.(models.Team)
	}
	renderTemplate(w, p)
}

func GetScores(w http.ResponseWriter, r *http.Request) {
	scores := DataGetAllScore()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		Logger.Error("Error encoding json: ", err)
	}
}

func GetScoresSplit(w http.ResponseWriter, r *http.Request) {
	scores := DataGetAllScoreSplitByType()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		Logger.Error("Error encoding json: ", err)
	}
}

func GetServices(w http.ResponseWriter, r *http.Request) {
	services := DataGetServiceList()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		Logger.Error("Error encoding json: ", err)
	}
}

func getChallenges() []ChallengeCount {
	totals, err := DataGetTotalChallenges()
	if err != nil {
		Logger.Error("Could not get challenges: ", err)
	}
	return totals
}

func getTeamChallenges(teamname string) []ChallengeCount {
	acquired, err := DataGetTeamChallenges(teamname)
	if err != nil {
		Logger.Error("Could not get team challenges: ", err)
	}
	return acquired
}

func getChallengesOwnerOf(adminof, teamgroup string) []string {
	switch teamgroup {
	case "admin", "blackteam":
		return DataGetChallengeGroupsList()
	default:
		return []string{adminof}
	}
}

func getTeamScore(teamname string) int {
	return DataGetTeamScore(teamname)
}

func getAllTeamScores() []map[string]interface{} {
	results := DataGetAllScoreSplitByType()
	scores := make([]map[string]interface{}, 0, len(results)/2)

	acc := make(map[string]map[string]interface{})
	for _, r := range results {
		score, ok := acc[r.Teamname]
		if ok {
			score[r.Type] = r.Points
			score["Points"] = score["CTF"].(int) + score["Service"].(int)
			scores = append(scores, score)
		} else {
			acc[r.Teamname] = map[string]interface{}{
				"Teamnumber": r.Teamnumber,
				"Teamname":   r.Teamname,
				r.Type:       r.Points,
			}
		}
	}

	return scores
}

func existsSpecialFlags() bool {
	return len(specialChallenges) > 0
}

func renderTemplate(w http.ResponseWriter, p Page) {
	err := templates[p.Title].ExecuteTemplate(w, p.Title+".tmpl", &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
