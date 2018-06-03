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
	tmpl, ok := templates[p.Title]
	if !ok {
		Logger.Errorln("Template does not exist:", p.Title)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.ExecuteTemplate(w, "base", &p)
	if err != nil {
		Logger.WithError(err).WithField("name", p.Title).Error("Failed to execute template")
	}
}

// Parse templates at startup
func ensureAppTemplates() {
	if templates != nil {
		return
	}

	templates = make(map[string]*template.Template)

	funcMap := buildHelperMap()
	includes := template.Must(template.New("base").Funcs(funcMap).ParseGlob("tmpl/includes/*.tmpl"))
	layouts := mustGlobFiles("tmpl/*.tmpl")

	for _, layout := range layouts {
		title := strings.TrimSuffix(filepath.Base(layout), ".tmpl")
		clone := template.Must(includes.Clone())
		templates[title] = template.Must(clone.ParseFiles(layout))
	}
}

func ShowHome(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	p := Page{Title: "homepage"}
	if t != nil {
		p.T = *t
	}
	renderTemplate(w, p)
}

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if getCtxTeam(r) == nil {
		p := Page{
			Title: "login",
		}
		renderTemplate(w, p)
	} else {
		http.Redirect(w, r, "/dashboard", 302)
	}
}

func SubmitLogin(w http.ResponseWriter, r *http.Request) {
	loggedIn := CheckCreds(w, r)
	if loggedIn {
		http.Redirect(w, r, "/dashboard", 302)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	err := sessionManager.Load(r).Destroy(w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		Logger.WithError(err).Error("Failed to logout user")
	}
	http.Redirect(w, r, "/login", 302)
}

func ShowTeamDashboard(w http.ResponseWriter, r *http.Request) {
	p := Page{
		Title: "dashboard",
		T:     *getCtxTeam(r),
	}
	renderTemplate(w, p)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	if t != nil {
		p := Page{
			Title: "challenges",
			T:     *t,
		}
		renderTemplate(w, p)
	}
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	p := Page{Title: "scoreboard"}
	if t != nil {
		p.T = *t
	}
	renderTemplate(w, p)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	p := Page{Title: "services"}
	if t != nil {
		p.T = *t
	}
	renderTemplate(w, p)
}
