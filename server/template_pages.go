package server

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pereztr5/cyboard/server/models"
)

type Page struct {
	Title string
	T     models.Team
}

var templates map[string]*template.Template

func renderTemplate(w http.ResponseWriter, p Page) {
	err := templates[p.Title].ExecuteTemplate(w, p.Title+".tmpl", &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Parse templates at startup
func ensureAppTemplates() {
	if templates != nil {
		return
	}

	templates = make(map[string]*template.Template)
	funcMap := buildHelperMap()
	includes := mustGlobFiles("tmpl/includes/*.tmpl")
	layouts := mustGlobFiles("tmpl/*.tmpl")

	for _, layout := range layouts {
		files := append(includes, layout)
		title := strings.TrimSuffix(filepath.Base(layout), ".tmpl")
		templates[title] = template.Must(template.New(layout).Funcs(funcMap).ParseFiles(files...))
	}
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
