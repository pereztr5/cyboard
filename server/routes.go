package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

type Page struct {
	Title string
	T     Team
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

func CreateWebRouter(teamScoreUpdater, servicesUpdater *broadcastHub) *mux.Router {
	router := mux.NewRouter()
	// Public Routes
	router.HandleFunc("/", ShowHome).Methods("GET")
	router.HandleFunc("/login", ShowLogin).Methods("GET")
	router.HandleFunc("/login", SubmitLogin).Methods("POST")
	router.HandleFunc("/logout", Logout).Methods("GET")
	router.HandleFunc("/scoreboard", ShowScoreboard).Methods("GET")
	router.HandleFunc("/team/services", ShowServices).Methods("GET")
	// Public API
	// TODO: Make this the name of AIS challenge
	router.HandleFunc("/team/scores", GetScores).Methods("GET")
	router.HandleFunc("/team/scores/split", GetScoresSplit).Methods("GET")
	router.HandleFunc("/team/scores/live", teamScoreUpdater.ServeWs()).Methods("GET")
	router.HandleFunc("/team/services/live", servicesUpdater.ServeWs()).Methods("GET")
	router.HandleFunc("/services", GetServices).Methods("GET")
	return router
}

func CreateTeamRouter() (router *mux.Router, blackTeamRouter *mux.Router, ctfConfigRouter *mux.Router) {
	router = mux.NewRouter()
	router.HandleFunc("/team/dashboard", ShowTeamDashboard).Methods("GET")
	router.HandleFunc("/challenges", ShowChallenges).Methods("GET")
	router.HandleFunc("/challenges/list", GetPublicChallenges).Methods("GET")
	router.HandleFunc("/challenges/verify", CheckFlag).Methods("POST")
	router.HandleFunc("/challenges/verify/all", CheckAllFlags).Methods("POST")
	router.HandleFunc("/ctf/breakdown/subs_per_flag", GetBreakdownOfSubmissionsPerFlag).Methods("GET")
	router.HandleFunc("/ctf/breakdown/teams_flags", GetEachTeamsCapturedFlags).Methods("GET")

	blackTeamRouter = router.PathPrefix("/black/").Subrouter()
	blackTeamRouter.HandleFunc("/team/bonus", GrantBonusPoints).Methods("POST")

	ctfConfigRouter = router.PathPrefix("/ctf/").Subrouter()
	ctfConfigRouter.HandleFunc("/breakdown/subs_per_flag", GetBreakdownOfSubmissionsPerFlag).Methods("GET")
	ctfConfigRouter.HandleFunc("/breakdown/teams_flags", GetEachTeamsCapturedFlags).Methods("GET")
	{
		r := ctfConfigRouter.Path("/flags").Subrouter()
		r.Methods("GET").HandlerFunc(GetConfigurableFlags)
		r.Methods("POST").HandlerFunc(AddFlags)
	}
	{
		r := ctfConfigRouter.Path("/flags/{flag}").Subrouter()
		r.Methods("GET").HandlerFunc(GetFlagByName)
		r.Methods("POST").HandlerFunc(AddFlag)
		r.Methods("PUT").HandlerFunc(UpdateFlag)
		r.Methods("DELETE").HandlerFunc(DeleteFlag)
	}

	return
}

func CreateAdminRouter() *mux.Router {
	router := mux.NewRouter()
	admin := router.PathPrefix("/admin/").Subrouter()
	admin.HandleFunc("/teams", GetAllUsers).Methods("GET")
	admin.HandleFunc("/teams/add", AddTeams).Methods("POST").Headers("Content-Type", "text/csv; charset=UTF-8")
	admin.HandleFunc("/team/update/{teamName}", UpdateTeam).Methods("PUT").Headers("Content-Type", "application/json; charset=UTF-8")
	admin.HandleFunc("/team/delete/{teamName}", DeleteTeam).Methods("DELETE")
	return router
}

func ShowHome(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "homepage"}
	if t != nil {
		p.T = t.(Team)
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
		http.Redirect(w, r, "/team/dashboard", 302)
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
		http.Redirect(w, r, "/team/dashboard", 302)
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
		T:     r.Context().Value("team").(Team),
	}
	renderTemplate(w, p)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	if t != nil {
		p := Page{
			Title: "challenges",
			T:     t.(Team),
		}
		renderTemplate(w, p)
	}
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "scoreboard"}
	if t != nil {
		p.T = t.(Team)
	}
	renderTemplate(w, p)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team")
	p := Page{Title: "services"}
	if t != nil {
		p.T = t.(Team)
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
