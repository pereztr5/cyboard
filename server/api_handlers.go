package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
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
		ErrInternal(err)
	}
	http.Redirect(w, r, "/login", 302)
}

func GetScores(w http.ResponseWriter, r *http.Request) {
	scores, err := models.TeamsScores(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetScores"))
		return
	}
	render.JSON(w, r, scores)
}

func GetServices(w http.ResponseWriter, r *http.Request) {
	services, err := models.AllServices(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetServices"))
		return
	}
	render.JSON(w, r, services)
}

func GetPublicChallenges(w http.ResponseWriter, r *http.Request) {
	chals, err := models.AllPublicChallenges(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetPublicChallenges"))
		return
	}
	render.JSON(w, r, chals)
}

func SubmitFlag(w http.ResponseWriter, r *http.Request) {
	guess := &models.ChallengeGuess{Flag: r.FormValue("flag"), Name: r.FormValue("challenge")}
	if guess.Flag == "" && guess.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	team := getCtxTeam(r)
	flagState, err := models.CheckFlagSubmission(db, r.Context(), team, guess)
	if err != nil {
		saveCtxErrMsgFields(r, M{"challenge_name": guess.Name, "team": team.Name})
		ErrInternal(err)
		return
	}
	render.JSON(w, r, flagState)
}

func GetAllTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := models.AllTeams(db)
	if err != nil {
		ErrInternal(err)
		return
	}
	render.JSON(w, r, teams)
}

type BlueTeamInsertRequest struct {
	*models.BlueTeamStore
	Password string `json:"password"` // Becomes the `Hash` column
}

func (btr BlueTeamInsertRequest) Bind(r *http.Request) error {
	if btr.BlueTeamStore == nil {
		return errors.New("missing both team `name` and `blueteam_ip` fields")
	} else if btr.Password == "" {
		return errors.Errorf("insert bluteams (team=%q): passwords must not be empty",
			btr.BlueTeamStore.Name)
	}

	btr.BlueTeamStore.Hash = nil
	hash, err := bcrypt.GenerateFromPassword([]byte(btr.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrapf(err, "insert blueteams (team=%q)", btr.BlueTeamStore.Name)
	}
	btr.BlueTeamStore.Hash = hash
	btr.Password = ""
	return nil
}

type BlueTeamInsertRequestSlice []BlueTeamInsertRequest

func (batch BlueTeamInsertRequestSlice) Bind(r *http.Request) error {
	return nil
}

func AddTeams(w http.ResponseWriter, r *http.Request) {
	batch := BlueTeamInsertRequestSlice{}
	if err := render.Bind(r, batch); err != nil {
		ErrInvalidRequest(err)
		return
	}

	newteams := make(models.BlueTeamStoreSlice, len(batch), len(batch))
	for idx, t := range batch {
		newteams[idx] = *t.BlueTeamStore
	}
	if err := newteams.Insert(db); err != nil {
		ErrInternal(err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type TeamUpdateRequest struct {
	*models.Team
	Password *string `json:"password"` // Becomes the `Hash` column
}

func (tr *TeamUpdateRequest) Bind(r *http.Request) error {
	if tr.Team == nil {
		return errors.New("missing required team fields")
	}

	tr.Team.Hash = nil
	if tr.Password != nil {
		if *tr.Password == "" {
			return errors.Errorf("update team (team=%q): passwords must not be empty",
				tr.Team.Name)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(*tr.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrapf(err, "update team (team=%q)", tr.Team.Name)
		}
		tr.Team.Hash = hash
		tr.Password = nil
	}
	return nil
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	op := &TeamUpdateRequest{}
	if err := render.Bind(r, op); err != nil {
		ErrInvalidRequest(err)
		return
	}
	teamID, err := strconv.Atoi(chi.URLParam(r, "teamID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "UpdateTeam, teamID URL param"))
		return
	} else if teamID != op.Team.ID {
		ErrInvalidRequest(errors.New("UpdateTeam (IDs do not match)"))
		return
	}

	if err := op.Team.Update(db); err != nil {
		ErrInternal(errors.Wrap(err, "UpdateTeam"))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.Atoi(chi.URLParam(r, "teamID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "DeleteTeam, teamID URL param"))
		return
	}

	team := &models.Team{ID: teamID}
	if err = team.Delete(db); err != nil {
		ErrInternal(err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type BonusPointsRequest struct {
	*models.OtherPoints
	TeamIDs []int `json:"teams"`
}

func GrantBonusPoints(w http.ResponseWriter, r *http.Request) {
	batch := &BonusPointsRequest{}
	if err := render.Decode(r, batch); err != nil {
		ErrInvalidRequest(err)
		return
	}

	cnt := len(batch.TeamIDs)
	bonus := make(models.OtherPointsSlice, cnt, cnt)
	for idx, teamID := range batch.TeamIDs {
		bonus[idx] = *batch.OtherPoints
		bonus[idx].TeamID = teamID
	}
	if err := bonus.Insert(db); err != nil {
		ErrInternal(err)
		return
	}

	CaptFlagsLogger.WithFields(logrus.Fields{
		"teams":  batch.TeamIDs,
		"reason": batch.Reason,
		"points": batch.Points,
	}).Infoln("Bonus awarded!")

	w.WriteHeader(http.StatusOK)
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
