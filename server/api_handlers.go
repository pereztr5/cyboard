package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := PingDB(r.Context()); err != nil {
		Logger.WithError(err).Errorf("PingHandler: DB is down")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Uh oh something's wrong!"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`🦆`))
	}
}

func GetScoresSplit(w http.ResponseWriter, r *http.Request) {
	scores, err := models.TeamsScores(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrapf(err, "GetScoresSplit (%s)", r.URL.Path))
		return
	}
	render.JSON(w, r, scores)
}

func GetServices(w http.ResponseWriter, r *http.Request) {
	services, err := models.AllServices(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrapf(err, "GetServices (%s)", r.URL.Path))
		return
	}
	render.JSON(w, r, services)
}

func GetPublicChallenges(w http.ResponseWriter, r *http.Request) {
	chals, err := models.AllPublicChallenges(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrapf(err, "GetPublicChallenges (%s)", r.URL.Path))
		return
	}
	render.JSON(w, r, chals)
}

func SubmitFlag(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	flag, challenge := r.FormValue("flag"), r.FormValue("challenge")
	if flag == "" && challenge == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var found FlagState
	var err error
	challengeQuery := models.Challenge{Flag: flag}
	if challenge != "" {
		challengeQuery.Name = challenge
	}
	found, err = DataCheckFlag(t, challengeQuery)
	if err != nil {
		Logger.Errorf("Error checking flag: %s for team: %s: %v", flag, t.Name, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, found)
}

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	users := DataGetAllUsers()
	err := json.NewEncoder(w).Encode(users)
	if err != nil {
		Logger.Error("Error with GetAllUsers: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func AddTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := ParseTeamCsv(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	// The unique indexes on the teams collection
	// will prevent bad duplicates.
	err = DataAddTeams(teams)
	if err != nil {
		// MongoDB errors should be safe to give back to admins.
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	teamName := chi.URLParam(r, "teamName")
	if teamName == "" {
		Logger.Error("Failed to update team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var updateOp map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateOp); err != nil {
		Logger.Error("Error decoding update PUT body: ", err)
		http.Error(w, err.Error(), 500)
		return
	}

	if err := DataUpdateTeam(teamName, updateOp); err != nil {
		Logger.Error("Failed to update team: ", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamName := chi.URLParam(r, "teamName")
	if teamName == "" {
		Logger.Error("Failed to delete team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err := DataDeleteTeam(teamName)
	if err != nil {
		Logger.Error("Failed to delete team: ", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type BonusDescriptor struct {
	Teams   []string `json:"teams"`
	Points  int      `json:"points"`
	Details string   `json:"details"`
}

func GrantBonusPoints(w http.ResponseWriter, r *http.Request) {
	var bonus BonusDescriptor

	if err := json.NewDecoder(r.Body).Decode(&bonus); err != nil {
		Logger.Error("GrantBonusPoints: failed to decode request body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := grantBonusPoints(bonus); err != nil {
		Logger.Errorln("GrantBonusPoints failed:", err)
		if err == mgo.ErrCursor {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func grantBonusPoints(bonus BonusDescriptor) error {
	if len(bonus.Teams) == 0 {
		return fmt.Errorf("Field 'teams' must be filled in")
	}

	teams := make([]models.Team, len(bonus.Teams))
	for i, teamName := range bonus.Teams {
		team, err := GetTeamByTeamname(teamName)
		if err != nil {
			return fmt.Errorf("failed to fetch team '%s': %v", teamName, err)
		}
		teams[i] = team
	}
	if len(teams) != len(bonus.Teams) {
		return fmt.Errorf("failed to generate complete list of teams (did you have duplicate team names?)")
	}

	now := time.Now()
	results := make([]models.Result, len(teams))
	for i := range results {
		results[i] = models.Result{
			Type:       CTF,
			Group:      "BONUS",
			Timestamp:  now,
			Teamname:   teams[i].Name,
			Teamnumber: teams[i].Number,
			Details:    bonus.Details,
			Points:     bonus.Points,
		}
	}
	CaptFlagsLogger.WithField("teams", bonus.Teams).WithField("challenge", bonus.Details).WithField("chalGroup", "BONUS").
		WithField("points", bonus.Points).Println("Bonus awarded!")

	return DataAddResults(results, false)
}

// CTF Configuration

// findConfigurableFlagFromReq will find the matching flag in the URL
// from the list of owned challenges that exist on the request context.
// (They are added by the RequireCtfGroupOwner middleware)
func findConfigurableFlagFromReq(r *http.Request) *models.Challenge {
	chals, flagName := getCtxOwnedChallenges(r), chi.URLParam(r, "flag")
	for _, c := range chals {
		if c.Name == flagName {
			return &c
		}
	}
	return nil
}

// ctfIsAdminOf returns true if the team is allowed control
// over the challenge.
func ctfIsAdminOf(t *models.Team, c *models.Challenge) bool {
	switch t.Group {
	case "admin", "blackteam":
		return true
	default:
		return t.AdminOf == c.Group
	}
}

func getChallengesOwnerOf(adminof, teamgroup string) []string {
	switch teamgroup {
	case "admin", "blackteam":
		return DataGetChallengeGroupsList()
	default:
		return []string{adminof}
	}
}

func GetConfigurableFlags(w http.ResponseWriter, r *http.Request) {
	chals := getCtxOwnedChallenges(r)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(chals); err != nil {
		Logger.Error("Error encoding GetConfigurableFlags json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func AddFlags(w http.ResponseWriter, r *http.Request) {
	team := getCtxTeam(r)
	var insertOp []models.Challenge

	if err := json.NewDecoder(r.Body).Decode(&insertOp); err != nil {
		Logger.Error("AddFlags: decode req body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := DataAddChallenges(team, insertOp); err != nil {
		Logger.Error("AddFlags:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func GetFlagByName(w http.ResponseWriter, r *http.Request) {
	chal := findConfigurableFlagFromReq(r)
	if chal == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(chal); err != nil {
		Logger.Error("GetFlagByName: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func AddFlag(w http.ResponseWriter, r *http.Request) {
	team := getCtxTeam(r)
	var insertOp models.Challenge

	if err := json.NewDecoder(r.Body).Decode(&insertOp); err != nil {
		Logger.Error("AddFlag: decode req body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else if chi.URLParam(r, "flag") != insertOp.Name {
		http.Error(w, "URL flag name and body's flag name must match", http.StatusBadRequest)
		return
	} else if !ctfIsAdminOf(team, &insertOp) {
		Logger.WithField("challenge", insertOp.Name).WithField("team", team.Name).Error("AddFlag: unauthorized to add flag")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if err := DataAddChallenge(&insertOp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func UpdateFlag(w http.ResponseWriter, r *http.Request) {
	chal := findConfigurableFlagFromReq(r)
	if chal == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	var updateOp models.Challenge
	if err := json.NewDecoder(r.Body).Decode(&updateOp); err != nil {
		Logger.Error("UpdateFlag: decode req body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := DataUpdateChallenge(&chal.Id, &updateOp); err != nil {
		Logger.Error("UpdateFlag: db update: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteFlag(w http.ResponseWriter, r *http.Request) {
	team, deleteOp := getCtxTeam(r), findConfigurableFlagFromReq(r)
	if deleteOp == nil {
		flagName := chi.URLParam(r, "flag")
		Logger.WithField("challenge", flagName).WithField("team", team.Name).Error("DeleteFlag: unauthorized")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	if err := DataDeleteChallenge(&deleteOp.Id); err != nil {
		Logger.Error("DeleteFlag: db remove: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// todo(tbutts): Reduce copied code. Particularly in the Breakdown methods, and anything that returns JSON.
// todo(tbutts): Consider a middleware or some abstraction on the Json encoding (gorilla may already provide this)

func GetBreakdownOfSubmissionsPerFlag(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	chalGroups := getChallengesOwnerOf(t.AdminOf, t.Group)

	flagsWithCapCounts, err := DataGetSubmissionsPerFlag(chalGroups)
	if err != nil {
		Logger.Error("Failed to get flags w/ occurences of capture: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(flagsWithCapCounts)
	if err != nil {
		Logger.Error("Error encoding FlagCaptures breakdown json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func GetEachTeamsCapturedFlags(w http.ResponseWriter, r *http.Request) {
	t := getCtxTeam(r)
	chalGroups := getChallengesOwnerOf(t.AdminOf, t.Group)

	teamsWithCapturedFlags, err := DataGetEachTeamsCapturedFlags(chalGroups)
	if err != nil {
		Logger.Error("Failed to get each teams' flag captures: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(teamsWithCapturedFlags)
	if err != nil {
		Logger.Error("Error encoding each teams' flag captures breakdown json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}
