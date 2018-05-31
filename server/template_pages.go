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
