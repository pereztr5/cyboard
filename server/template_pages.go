package server

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/sirupsen/logrus"
)

type Page struct {
	File  string       // Template file (with suffix trimmed)
	Title string       // Visible page name
	T     *models.Team // Viewer Context
	Error error        // Cause, if any, of rendering failure

	Data map[string]interface{} // Page-specific data
}

func getPage(r *http.Request, templateFile, title string) *Page {
	team := getCtxTeam(r)
	page := &Page{File: templateFile, Title: title}
	if team != nil {
		page.T = team
	}
	return page
}

func (p *Page) checkErr(err error, target string) {
	if err != nil {
		Logger.WithError(err).WithFields(logrus.Fields{
			"team":   p.T.Name,
			"file":   p.File,
			"target": target,
		}).Error("unable to get data to render page")

		if p.Error == nil {
			p.Error = err
		}
	}
}

var templates map[string]*template.Template

func renderTemplate(w http.ResponseWriter, p *Page) {
	tmpl, ok := templates[p.File]
	if !ok {
		Logger.Errorln("Template does not exist:", p.File)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.ExecuteTemplate(w, "base", &p)
	if err != nil {
		Logger.WithError(err).WithField("name", p.File).Error("Failed to execute template")
	}
}

// Parse templates at startup
func ensureAppTemplates() {
	if templates != nil {
		return
	}

	templates = make(map[string]*template.Template)

	funcMap := buildHelperMap()
	includes := template.Must(template.New("base").Funcs(funcMap).ParseGlob("ui/tmpl/includes/*.tmpl"))
	layouts := mustGlobFiles("ui/tmpl/*.tmpl")

	for _, layout := range layouts {
		title := strings.TrimSuffix(filepath.Base(layout), ".tmpl")
		clone := template.Must(includes.Clone())
		templates[title] = template.Must(clone.ParseFiles(layout))
	}
}

func ShowHome(w http.ResponseWriter, r *http.Request) {
	if time.Now().After(appCfg.Event.Start) {
		page := getPage(r, "homepage", "Homepage")
		page.Data = M{"Video": getHomepageVid()}
		renderTemplate(w, page)
	} else {
		// Before the event has started, show a timer counting down
		countdownTmpl := templates["countdown"]
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Don't trust the user's browser to have time configured correctly. Use a duration
		// instead of a datetime, to act as a monotonic time keeping method.
		err := countdownTmpl.ExecuteTemplate(w, "countdown", time.Until(appCfg.Event.Start))
		if err != nil {
			Logger.WithError(err).WithField("name", "countdown").Error("Failed to execute template")
		}
	}
}

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	if getCtxTeam(r) == nil {
		page := &Page{File: "login", Title: "Login"}
		renderTemplate(w, page)
	} else {
		http.Redirect(w, r, "/dashboard", 302)
	}
}

func ShowTeamDashboard(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "dashboard", "Dashboard")
	team := page.T
	page.Data = make(map[string]interface{})

	var err error
	page.Data["ctfProgress"], err = models.GetTeamCTFProgress(db, team.ID)
	page.checkErr(err, "ctf progress")

	renderTemplate(w, page)
}

func ShowChallenges(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "challenges", "Challenges")

	team := getCtxTeam(r)
	chals, err := models.AllPublicChallenges(db, team.ID)
	page.checkErr(err, "public challenges")
	page.Data = M{"GroupsOfChallenges": chals}
	renderTemplate(w, page)
}

func ShowScoreboard(w http.ResponseWriter, r *http.Request) {
	var err error
	var page *Page
	if _, ok := r.URL.Query()["noscript"]; !ok {
		page = getPage(r, "scoreboard", "Scoreboard")
		page.Data = make(map[string]interface{})
	} else {
		page = getPage(r, "noscript_scoreboard", "Scoreboard")
		page.Data = make(map[string]interface{})

		page.Data["TeamsScores"], err = models.TeamsScores(db)
		page.checkErr(err, "team scores")
	}

	page.Data["Teams"], err = models.AllBlueteams(db)
	page.checkErr(err, "all blue teams")

	page.Data["Statuses"], err = models.TeamServiceStatuses(db)
	page.checkErr(err, "all teams' service statuses")

	renderTemplate(w, page)
}

func ShowServices(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "services", "Services")
	page.Data = make(map[string]interface{})

	page.Data["Teams"], err = models.AllBlueteams(db)
	page.checkErr(err, "all blue teams")

	page.Data["Statuses"], err = models.TeamServiceStatuses(db)
	page.checkErr(err, "all teams' service statuses")

	renderTemplate(w, page)
}

/* CTF Creator pages */

func ShowCtfConfig(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "staff_ctf_cfg", "Admin CTF")

	chals, err := models.AllChallenges(db)
	page.checkErr(err, "all challenges")
	page.Data = M{"Challenges": chals, "TotalPoints": models.ChallengeSlice(chals).Sum()}
	renderTemplate(w, page)
}

func ShowCtfDashboard(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "staff_ctf_dash", "CTF Statistics")
	page.Data = make(map[string]interface{})

	page.Data["ChallengeCapturesPerFlag"], err = models.ChallengeCapturesPerFlag(db)
	page.checkErr(err, "challenge captures per flag")

	page.Data["ChallengeCapturesPerTeam"], err = models.ChallengeCapturesPerTeam(db)
	page.checkErr(err, "challenge captures per team")

	renderTemplate(w, page)
}

/* Admin Pages */

func ShowTeamsConfig(w http.ResponseWriter, r *http.Request) {
	page := getPage(r, "admin_teams_cfg", "Admin Teams")

	teams, err := models.AllTeams(db)
	page.checkErr(err, "all teams")
	page.Data = M{"Teams": teams}
	renderTemplate(w, page)
}

func ShowServicesConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "admin_services_cfg", "Admin Services")
	page.Data = make(map[string]interface{})

	services, err := models.AllServices(db)
	page.checkErr(err, "all services")
	page.Data["Services"] = services
	page.Data["TotalPoints"] = models.ServiceSlice(services).Sum()
	page.Data["ScriptFiles"], err = getFileList(ScriptMgr.pathBuilder(r))
	page.checkErr(err, "script files")

	page.Data["Event"] = appCfg.Event
	page.Data["ServiceMonitor"] = appCfg.ServiceMonitor

	renderTemplate(w, page)
}

func ShowServiceScriptsConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "admin_services_scripts", "Check Scripts")
	page.Data = make(map[string]interface{})

	page.Data["ScriptFiles"], err = getFileList(ScriptMgr.pathBuilder(r))
	page.checkErr(err, "script files")

	renderTemplate(w, page)
}

func ShowBonusPage(w http.ResponseWriter, r *http.Request) {
	var err error
	page := getPage(r, "staff_bonus", "Bonuses")
	page.Data = make(map[string]interface{})

	page.Data["Blueteams"], err = models.AllBlueteams(db)
	page.checkErr(err, "all blue teams")

	renderTemplate(w, page)
}
