package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/jackc/pgx"
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

func GetServicesStatuses(w http.ResponseWriter, r *http.Request) {
	services, err := models.TeamServiceStatuses(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetServicesStatuses"))
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
		if err == pgx.ErrNoRows {
			CaptFlagsLogger.WithFields(logrus.Fields{"team": team.Name, "guess": guess.Flag, "challenge": guess.Name}).Println("Bad guess")
		} else {
			saveCtxErrMsgFields(r, M{"challenge_name": guess.Name, "team": team.Name})
			ErrInternal(err)
			return
		}
	}
	if flagState == models.ValidFlag {
		CaptFlagsLogger.WithFields(logrus.Fields{"team": team.Name, "challenge": guess.Name, "category": guess.Category}).Println("Score!!")
	}
	render.JSON(w, r, flagState)
}

func GetAllTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := models.AllTeams(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetAllTeams"))
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
	w.WriteHeader(http.StatusCreated)
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

	w.WriteHeader(http.StatusCreated)
}

// CTF Configuration

func GetAllFlags(w http.ResponseWriter, r *http.Request) {
	challenges, err := models.AllChallenges(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetAllFlags"))
		return
	}
	// NOTE: render.JSON will automatically escape any HTML during json encoding,
	// which we may not want to do if we decide that Challenge.Body could have raw HTML.
	render.JSON(w, r, challenges)
}

func AddFlags(w http.ResponseWriter, r *http.Request) {
	newChallenges := models.ChallengeSlice{}
	if err := render.Decode(r, newChallenges); err != nil {
		ErrInvalidRequest(err)
		return
	}

	if err := newChallenges.Insert(db); err != nil {
		ErrInternal(err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func GetFlagByID(w http.ResponseWriter, r *http.Request) {
	flagID, err := strconv.Atoi(chi.URLParam(r, "flagID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "GetFlagByName, flagID URL param"))
		return
	}

	challenge, err := models.ChallengeByID(db, flagID)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetFlagByName"))
		return
	}
	render.JSON(w, r, challenge)
	w.WriteHeader(http.StatusOK)
}

func UpdateFlag(w http.ResponseWriter, r *http.Request) {
	challenge := &models.Challenge{}
	if err := render.Decode(r, challenge); err != nil {
		ErrInvalidRequest(err)
		return
	}
	flagID, err := strconv.Atoi(chi.URLParam(r, "flagID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "UpdateFlag, flagID URL param"))
		return
	} else if flagID != challenge.ID {
		ErrInvalidRequest(errors.New("UpdateFlag (IDs do not match)"))
		return
	}

	if err := challenge.Update(db); err != nil {
		ErrInternal(err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteFlag(w http.ResponseWriter, r *http.Request) {
	flagID, err := strconv.Atoi(chi.URLParam(r, "flagID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "DeleteFlag, flagID URL param"))
		return
	}

	challenge := &models.Challenge{ID: flagID}
	if err := challenge.Delete(db); err != nil {
		ErrInternal(err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Service Configuration

func GetAllServices(w http.ResponseWriter, r *http.Request) {
	services, err := models.AllServices(db)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetAllServices"))
		return
	}
	render.JSON(w, r, services)
}

func GetService(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.Atoi(chi.URLParam(r, "serviceID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "GetService, serviceID URL param"))
		return
	}

	service, err := models.ServiceByID(db, serviceID)
	if err != nil {
		RenderQueryErr(w, r, errors.Wrap(err, "GetService"))
		return
	}
	render.JSON(w, r, service)
	w.WriteHeader(http.StatusOK)
}

type ServiceRequest struct {
	*models.Service
}

func (sr *ServiceRequest) Bind(r *http.Request) error {
	if _, ok := r.URL.Query()["rawpoints"]; !ok {
		pts := CalcPointsPerCheck(sr.Service, &appCfg.Event, appCfg.ServiceMonitor.Intervals)
		sr.Points = &pts
	} else if sr.Points == nil {
		return errors.New(`request with ?rawpoints=true must have {"points": <decimal>} field`)
	}
	return nil
}

func AddService(w http.ResponseWriter, r *http.Request) {
	srv := new(ServiceRequest)
	if err := render.Bind(r, srv); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := srv.Service.Insert(db); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func UpdateService(w http.ResponseWriter, r *http.Request) {
	srv := new(ServiceRequest)
	if err := render.Bind(r, srv); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := srv.Service.Update(db); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteService(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.Atoi(chi.URLParam(r, "serviceID"))
	if err != nil {
		ErrInvalidRequest(errors.Wrap(err, "DeleteService, serviceID URL param"))
		return
	}

	service := &models.Service{ID: serviceID}
	if err := service.Delete(db); err != nil {
		ErrInternal(err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Scoring graphs

func GetBreakdownOfSubmissionsPerFlag(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerFlag(db)
	if err != nil {
		ErrInternal(err)
		return
	}
	render.JSON(w, r, brkdwn)
}

func GetEachTeamsCapturedFlags(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerTeam(db)
	if err != nil {
		ErrInternal(err)
		return
	}
	render.JSON(w, r, brkdwn)
}
