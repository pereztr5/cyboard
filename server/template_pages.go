package server

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/sirupsen/logrus"
)

type Page struct {
	Title string
	T     *models.Team
	Error error
	Data  map[string]interface{}
}

func getPage(r *http.Request, title string) *Page {
	team := getCtxTeam(r)
	page := &Page{Title: title}
	if team != nil {
		page.T = team
	}
	return page
}

func (p *Page) checkErr(err error, target string) {
	if err != nil {
		Logger.WithError(err).WithFields(logrus.Fields{
			"team":   p.T.Name,
			"title":  p.Title,
			"target": target,
		}).Error("unable to get data to render page")
		p.Error = err
	}
}

var templates map[string]*template.Template

func renderTemplate(w http.ResponseWriter, p *Page) {
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
	page := getPage(r, "homepage")
	renderTemplate(w, page)
}

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if getCtxTeam(r) == nil {
		page := &Page{
			Title: "login",
		}
		renderTemplate(w, page)
	} else {
		http.Redirect(w, r, "/dashboard", 302)
	}
}

func ShowTeamDashboard(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "dashboard")
	team := page.T
	page.Data = make(map[string]interface{})

	var err error
	if team.RoleName == models.TeamRoleBlueteam {
		page.Data["ctfProgress"], err = models.GetTeamCTFProgress(db, team.ID)
		page.checkErr(err, "ctf progress")
	} else {
		// TODO: Admin panel.
		// build up admin dashboard model data
	}

	renderTemplate(w, page)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "challenges")

	chals, err := models.AllPublicChallenges(db)
	page.checkErr(err, "public challenges")
	page.Data = M{"Challenges": chals}
	renderTemplate(w, page)
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "scoreboard")

	teamsScores, err := models.TeamsScores(db)
	page.checkErr(err, "team scores")
	page.Data = M{"TeamsScores": teamsScores}

	renderTemplate(w, page)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "services")
	page.Data = make(map[string]interface{})

	page.Data["Teams"], err = models.AllBlueteams(db)
	page.checkErr(err, "all blue teams")

	page.Data["Statuses"], err = models.TeamServiceStatuses(db)
	page.checkErr(err, "all teams' service statuses")

	renderTemplate(w, page)
}
