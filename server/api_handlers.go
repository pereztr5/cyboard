package server

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-chi/render"
	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// ApiQuery is a helper for HTTP GET operations for most models.
// It handles responding to the API user with the results of a db query,
// and handling of non-nil errors if they arise.
func ApiQuery(w http.ResponseWriter, r *http.Request, v interface{}, err error) {
	if err != nil {
		RenderQueryErr(w, r, err)
		return
	}
	render.JSON(w, r, v)
}

// ApiCreate is a helper for HTTP POST operations for ~a few~ models.
//
// model `v` is expected to implement either `models.Inserter` or `models.ManyInserter`,
// or else it will error.
//
// This helper less reusable than the other helpers, due to the extra type gymnastics
// which are difficult to abstract away (transforming input json into completely
// different types, arranging slice layouts, etc.)
func ApiCreate(w http.ResponseWriter, r *http.Request, v interface{}) {
	var err error
	if vbind, ok := v.(render.Binder); ok {
		err = render.Bind(r, vbind)
	} else {
		err = render.Decode(r, v)
	}

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	switch op := v.(type) {
	case models.Inserter:
		err = op.Insert(db)
	case models.ManyInserter:
		err = op.Insert(db)
	default:
		render.Render(w, r, ErrInternal(fmt.Errorf(
			"model does not implement an insert-like method: type=%T", v)))
		return
	}

	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// ApiUpdate is a helper for HTTP UPDATE operations for most models.
// The model is expected to have an integer `ID` field, or it will panic.
func ApiUpdate(w http.ResponseWriter, r *http.Request, v models.Updater) {
	var err error
	if vbind, ok := v.(render.Binder); ok {
		err = render.Bind(r, vbind)
	} else {
		err = render.Decode(r, v)
	}

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	id := getCtxIdParam(r)
	reflect.ValueOf(v).Elem().FieldByName(`ID`).SetInt(int64(id))

	if err := v.Update(db); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	render.JSON(w, r, v)
}

// ApiDelete is a helper for HTTP DELETE operations for most models.
// The model is expected to have an integer `ID` field, or it will panic.
func ApiDelete(w http.ResponseWriter, r *http.Request, v models.Deleter) {
	id := getCtxIdParam(r)
	reflect.ValueOf(v).Elem().FieldByName(`ID`).SetInt(int64(id))

	if err := v.Delete(db); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

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
	if r.FormValue(formCredsTeam) == "" || r.FormValue(formCredsPass) == "" {
		render.Render(w, r, ErrInvalidBecause(fmt.Sprintf("Missing form fields: "+
			"requires both '%s' and '%s'", formCredsTeam, formCredsPass)))
	}

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
		render.Render(w, r, ErrInternal(err))
	}
	http.Redirect(w, r, "/login", 302)
}

func GetScores(w http.ResponseWriter, r *http.Request) {
	scores, err := models.TeamsScores(db)
	ApiQuery(w, r, scores, err)
}

func GetServicesStatuses(w http.ResponseWriter, r *http.Request) {
	services, err := models.TeamServiceStatuses(db)
	ApiQuery(w, r, services, err)
}

func GetPublicChallenges(w http.ResponseWriter, r *http.Request) {
	chals, err := models.AllPublicChallenges(db)
	ApiQuery(w, r, chals, err)
}

func SubmitFlag(w http.ResponseWriter, r *http.Request) {
	guess := &models.ChallengeGuess{Flag: r.FormValue("flag"), Name: r.FormValue("challenge")}
	if guess.Flag == "" {
		render.Render(w, r, ErrInvalidBecause(`Missing form field: 'flag'`))
		return
	}

	team := getCtxTeam(r)
	flagState, err := models.CheckFlagSubmission(db, r.Context(), team, guess)
	if err != nil {
		if err == pgx.ErrNoRows {
			CaptFlagsLogger.WithFields(logrus.Fields{"team": team.Name, "guess": guess.Flag, "challenge": guess.Name}).Println("Bad guess")
		} else {
			saveCtxErrMsgFields(r, M{"challenge_name": guess.Name, "team": team.Name})
			render.Render(w, r, ErrInternal(err))
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
	ApiQuery(w, r, teams, err)
}

func GetTeamByID(w http.ResponseWriter, r *http.Request) {
	id := getCtxIdParam(r)
	team, err := models.TeamByID(db, id)
	ApiQuery(w, r, team, err)
}

type BlueTeamInsertRequest struct {
	*models.BlueTeamStore
	Password string `json:"password,omitempty"` // Becomes the `Hash` column
}

func (btr *BlueTeamInsertRequest) Bind(r *http.Request) error {
	if btr.BlueTeamStore == nil {
		return errors.New(`missing all fields: 'name', 'password', 'blueteam_ip'`)
	} else if btr.Name == "" {
		return errors.New(`empty field: 'name'`)
	} else if btr.Password == "" {
		return errors.New(`empty field: 'password'`)
	} else if btr.BlueteamIP == 0 {
		return errors.New(`empty/zero field: 'blueteam_ip'`)
	}

	btr.BlueTeamStore.Hash = nil
	hash, err := bcrypt.GenerateFromPassword([]byte(btr.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	btr.BlueTeamStore.Hash = hash
	btr.Password = ""
	return nil
}

type BlueTeamInsertRequestSlice []BlueTeamInsertRequest

func (batch *BlueTeamInsertRequestSlice) Bind(r *http.Request) error {
	var err error
	xs := *batch
	for idx := range xs {
		if err = xs[idx].Bind(r); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("team [%d]", idx))
		}
	}
	return nil
}

func (batch BlueTeamInsertRequestSlice) Insert(tx models.TXer) error {
	newteams := make(models.BlueTeamStoreSlice, len(batch), len(batch))
	for idx, t := range batch {
		newteams[idx] = *t.BlueTeamStore
	}
	return newteams.Insert(tx)
}

func AddBlueteams(w http.ResponseWriter, r *http.Request) {
	batch := &BlueTeamInsertRequestSlice{}
	ApiCreate(w, r, batch)
}

type TeamUpdateRequest struct {
	*models.Team
	Password *string `json:"password,omitempty"` // Becomes the `Hash` column
}

func (tr *TeamUpdateRequest) Bind(r *http.Request) error {
	if tr.Team == nil {
		return errors.New("missing required team fields")
	}

	tr.Team.Hash = nil
	if tr.Password != nil {
		// If a password is specified, it at least can't be empty
		if *tr.Password == "" {
			return errors.New("empty field: 'password'")
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(*tr.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		tr.Team.Hash = hash
		tr.Password = nil
	}
	return nil
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	team := &TeamUpdateRequest{}
	ApiUpdate(w, r, team)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	team := &models.Team{}
	ApiDelete(w, r, team)
}

type BonusPointsRequest struct {
	*models.OtherPoints
	TeamIDs []int `json:"teams"`
}

func GrantBonusPoints(w http.ResponseWriter, r *http.Request) {
	batch := &BonusPointsRequest{}
	if err := render.Decode(r, batch); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	cnt := len(batch.TeamIDs)
	bonus := make(models.OtherPointsSlice, cnt, cnt)
	for idx, teamID := range batch.TeamIDs {
		bonus[idx] = *batch.OtherPoints
		bonus[idx].TeamID = teamID
	}
	if err := bonus.Insert(db); err != nil {
		render.Render(w, r, ErrInternal(err))
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
	// NOTE: ApiQuery will automatically escape any HTML during json encoding,
	// which we may not want to do if we decide that Challenge.Body could have raw HTML.
	ApiQuery(w, r, challenges, err)
}

func AddFlags(w http.ResponseWriter, r *http.Request) {
	newChallenges := &models.ChallengeSlice{}
	ApiCreate(w, r, newChallenges)
}

func GetFlagByID(w http.ResponseWriter, r *http.Request) {
	flagID := getCtxIdParam(r)
	challenge, err := models.ChallengeByID(db, flagID)
	ApiQuery(w, r, challenge, err)
}

func UpdateFlag(w http.ResponseWriter, r *http.Request) {
	challenge := &models.Challenge{}
	ApiUpdate(w, r, challenge)
}

func DeleteFlag(w http.ResponseWriter, r *http.Request) {
	challenge := &models.Challenge{}
	ApiDelete(w, r, challenge)
}

// Service Configuration

func GetAllServices(w http.ResponseWriter, r *http.Request) {
	services, err := models.AllServices(db)
	ApiQuery(w, r, services, err)
}

func GetServiceByID(w http.ResponseWriter, r *http.Request) {
	serviceID := getCtxIdParam(r)
	service, err := models.ServiceByID(db, serviceID)
	ApiQuery(w, r, service, err)
}

type ServiceRequest struct {
	*models.Service
}

func (sr *ServiceRequest) Bind(r *http.Request) error {
	if sr.Service == nil {
		return errors.New(`missing required 'service' fields`)
	}

	if _, ok := r.URL.Query()["rawpoints"]; !ok {
		pts := CalcPointsPerCheck(sr.Service, &appCfg.Event, appCfg.ServiceMonitor.Intervals)
		sr.Points = &pts
	} else if sr.Points == nil {
		return errors.New(`request with ?rawpoints=true must have {"points": <decimal>} field`)
	}

	return nil
}

func AddService(w http.ResponseWriter, r *http.Request) {
	srv := &ServiceRequest{}
	ApiCreate(w, r, srv)
}

func UpdateService(w http.ResponseWriter, r *http.Request) {
	srv := &ServiceRequest{}
	ApiUpdate(w, r, srv)
}

func DeleteService(w http.ResponseWriter, r *http.Request) {
	service := &models.Service{}
	ApiDelete(w, r, service)
}

// Scoring graphs

func GetBreakdownOfSubmissionsPerFlag(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerFlag(db)
	ApiQuery(w, r, brkdwn, err)
}

func GetEachTeamsCapturedFlags(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerTeam(db)
	ApiQuery(w, r, brkdwn, err)
}
