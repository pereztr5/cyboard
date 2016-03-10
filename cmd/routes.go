package cmd

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

// Logger is used to send logging messages to stdout.
var Logger = log.New(os.Stdout, " ", log.Ldate|log.Ltime|log.Lshortfile)

type Page struct {
	Title string
	T     Team
}

var templates map[string]*template.Template

// Parse templates at startup
// TODO Loop through all templates in directory
func init() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	funcMap := template.FuncMap{
		"getFlags": GetTeamFlags,
		"hasFlag":  TeamHasFlag,
	}

	templates["login"] = template.Must(template.ParseFiles("tmpl/header.tmpl", "tmpl/login.tmpl", "tmpl/footer.tmpl"))
	templates["teampage"] = template.Must(template.ParseFiles("tmpl/header.tmpl", "tmpl/teampage.tmpl", "tmpl/footer.tmpl"))
	t := template.New("flags")
	t.Funcs(funcMap)
	templates["flags"] = template.Must(t.ParseFiles("tmpl/header.tmpl", "tmpl/flags.tmpl", "tmpl/footer.tmpl"))
}

func CreateWebRouter() *mux.Router {
	router := mux.NewRouter()
	// Public Routes
	router.HandleFunc("/login", ShowLogin).Methods("GET")
	router.HandleFunc("/login", SubmitLogin).Methods("POST")
	router.HandleFunc("/logout", Logout)
	router.HandleFunc("/showflags", ShowFlags).Methods("GET")
	// Public API
	router.HandleFunc("/flags", GetFlags).Methods("GET")
	//router.HandleFunc("/scores", Score)
	return router
}

func CreateTeamRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/teampage", TeamPage)
	router.HandleFunc("/flags/verify", CheckFlag).Methods("POST")
	return router
}

/*
func GetScores(w http.ResponseWriter, r *http.Request) {
	scores, err := DataGetTeamScores()
	if err != nil {
		Logger.Printf("Error getting Team scores: %v\n", err)
	}
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		Logger.Printf("Error encoding json: %v\n", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}
*/

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if context.Get(r, "team") == nil {
		p := Page{
			Title: "login",
		}
		renderTemplate(w, p)
	} else {
		http.Redirect(w, r, "/teampage", 302)
	}
}

func SubmitLogin(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		log.Printf("Getting from Store failed: %v", err)
		http.Error(w, http.StatusText(400), 400)
		return
	}

	succ := CheckCreds(w, r)

	if succ {
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		http.Redirect(w, r, "/teampage", 302)
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

func TeamPage(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Title: "teampage",
		T:     context.Get(r, "team").(Team),
	}
	renderTemplate(w, p)
}

func ShowFlags(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team")
	if t != nil {
		p := Page{
			Title: "flags",
			T:     t.(Team),
		}
		renderTemplate(w, p)
	} else {
		http.Redirect(w, r, "/flags.html", 302)
	}
}

func GetTeamFlags() []Flag {
	flags, err := DataGetFlags()
	if err != nil {
		Logger.Printf("Error getting Flags: %v\n", err)
	}
	return flags
}

func TeamHasFlag(teamFlags []Flag) map[string]bool {
	flagMap := make(map[string]bool)
	for _, f := range teamFlags {
		flagMap[f.Flagname] = true
	}
	return flagMap
}

func renderTemplate(w http.ResponseWriter, p Page) {
	err := templates[p.Title].ExecuteTemplate(w, p.Title+".tmpl", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
