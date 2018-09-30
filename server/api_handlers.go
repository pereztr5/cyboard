package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Reusable API functions
//
// Covers HTTP GET, POST, PUT, and DELETE for most models.

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

// General Public API Methods (login, health check, public GETs)

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

// Blueteam API methods (view & submit challenges)

func GetPublicChallenges(w http.ResponseWriter, r *http.Request) {
	team := getCtxTeam(r)
	chals, err := models.AllPublicChallenges(db, team.ID)
	ApiQuery(w, r, chals, err)
}

func GetChallengeDescription(w http.ResponseWriter, r *http.Request) {
	flagID := getCtxIdParam(r)
	desc, err := models.GetPublicChallengeDescription(db, flagID)

	if err != nil {
		RenderQueryErr(w, r, err)
		return
	}
	render.PlainText(w, r, desc)
}

func SubmitFlag(w http.ResponseWriter, r *http.Request) {
	guess := &models.ChallengeGuess{Flag: r.FormValue("flag"), Name: r.FormValue("challenge")}
	if guess.Flag == "" {
		render.Render(w, r, ErrInvalidBecause(`Missing form field: 'flag'`))
		return
	}
	anon := guess.Name == ""

	team := getCtxTeam(r)
	logFields := logrus.Fields{"challenge": guess.Name, "guess": guess.Flag, "team": team.Name}
	if anon {
		logFields["challenge"] = "<anonymous>"
	}

	flagState, err := models.CheckFlagSubmission(db, r.Context(), team, guess)
	if err != nil {
		if err == pgx.ErrNoRows {
			CaptFlagsLogger.WithFields(logFields).Println("Bad guess")
		} else {
			saveCtxErrMsgFields(r, M(logFields))
			render.Render(w, r, ErrInternal(err))
			return
		}
	}
	if flagState == models.ValidFlag {
		logFields["challenge"] = guess.Name    // guess.Name is filled by models.CheckFlagSubmission on success
		logFields["category"] = guess.Category // Same deal with guess.Category
		logFields["anon"] = anon               // Mark whether this was an anonymous challenge
		delete(logFields, "guess")             // But don't need the correct guesses in the log file
		CaptFlagsLogger.WithFields(logFields).Println("Score!!")
	}
	render.JSON(w, r, flagState)
}

// User/Team management (admin-only):
//
// * get team configurations from the db
// * add several blue teams at once via JSON
// * add one staff team via JSON
// * update/delete teams
// JSON "password" fields will be saved as a hash+salt, and can never be retrieved.

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

type TeamModRequest struct {
	*models.Team
	Password *string `json:"password,omitempty"` // Becomes the `Hash` column
}

// Bind satisfies the go-chi/render#Binder interface, which does post-decode
// validation & transforms on JSON/XML request bodies.
//
// For a TeamModRequest, "PUT" reqs are not required to update the password/hash
// of the team, because it is impossible to recover.
func (tr *TeamModRequest) Bind(r *http.Request) error {
	if tr.Team == nil {
		return errors.New("missing required team fields: 'name', 'role_name'")
	} else if tr.Name == "" {
		return errors.New(`empty field: 'name'`)
	} else if tr.RoleName == models.TeamRoleUnspecified {
		return errors.New(`empty field: 'role_name'`)
	}
	tr.Team.Hash = nil

	hasPW := tr.Password != nil

	switch r.Method {
	default:
		// POST: must specify password
		if !hasPW || *tr.Password == "" {
			return errors.New("empty field: 'password'")
		}
	case "PUT":
		// PUT: If a password is specified, it at least can't be empty
		if hasPW && *tr.Password == "" {
			return errors.New("empty field: 'password'")
		}
	}

	if hasPW {
		hash, err := bcrypt.GenerateFromPassword([]byte(*tr.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		tr.Team.Hash = hash
		tr.Password = nil
	}
	return nil
}

func AddTeam(w http.ResponseWriter, r *http.Request) {
	team := &TeamModRequest{}
	ApiCreate(w, r, team)
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	team := &TeamModRequest{}
	ApiUpdate(w, r, team)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	team := &models.Team{}
	ApiDelete(w, r, team)
}

// Bonus Points API
// Allows arbitrary point awards/deductions, stored in the "other_points" db table.
// JSON input should give a list of team ids, point value, and a reason string.
//
// There is no formal way to `undo` these, just do another bonus with
// the negated point value and the reason "reverting bonus because of 'thing'".

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

	now := time.Now()
	cnt := len(batch.TeamIDs)
	bonus := make(models.OtherPointsSlice, cnt, cnt)
	for idx, teamID := range batch.TeamIDs {
		bonus[idx] = *batch.OtherPoints
		bonus[idx].TeamID = teamID
		bonus[idx].CreatedAt = now
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

// CTF File Management
// Files served with a given ctf challenge. e.g. `crackme` binaries, encrypted messages, etc.

var CtfFileMgr = FSContentManager{
	maxSize: 32 << 20, // Accept up to 32MB files
	mode:    0644,
	pathBuilder: func(r *http.Request) string {
		base := appCfg.Server.CtfFileDir
		ctfID := strconv.Itoa(getCtxIdParam(r))
		return filepath.Join(base, ctfID)
	},
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

// Service Monitor Check Scripts Configuration
//
// Uses an ancient storage API know as 'the filesystem' to save scripts/binaries,
// which are run by the service monitor.
// Supporting files (tables, keys) may also be managed through these endpoints.
// Everything goes in checks_dir (./scripts by default)
//

type FileInfo struct {
	Name    string    `json:"name"`     // base name of the file
	Size    int64     `json:"size"`     // length in bytes
	ModTime time.Time `json:"mod_time"` // modification time
}

type FSContentManager struct {
	maxSize     int64
	mode        os.FileMode
	pathBuilder func(*http.Request) string
}

func (cm FSContentManager) GetFileList(w http.ResponseWriter, r *http.Request) {
	fis, err := ioutil.ReadDir(cm.pathBuilder(r))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	infos := make([]FileInfo, len(fis), len(fis))
	for i, fi := range fis {
		infos[i] = FileInfo{Name: fi.Name(), Size: fi.Size(), ModTime: fi.ModTime()}
	}

	render.JSON(w, r, infos)
}

func (cm FSContentManager) GetFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	file, err := http.Dir(cm.pathBuilder(r)).Open(name)
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	} else if info.IsDir() {
		render.Render(w, r, ErrInvalidRequest(errors.New("path is a directory")))
		return
	}

	http.ServeContent(w, r, name, info.ModTime(), file)
}

func (cm FSContentManager) SaveFile(w http.ResponseWriter, r *http.Request) {
	dir := cm.pathBuilder(r)

	if _, err := os.Stat(dir); err != nil {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			render.Render(w, r, ErrInternal(
				errors.WithMessage(err, "unable to make internal content dir")))
			return
		}
	}

	err := r.ParseMultipartForm(cm.maxSize)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	fhs := r.MultipartForm.File["upload"]
	if len(fhs) == 0 {
		render.Render(w, r, ErrInvalidBecause("nothing uploaded"))
		return
	}

	processFile := func(fh *multipart.FileHeader) error {
		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer f.Close()
		path := filepath.Join(dir, fh.Filename)
		if stat, err := os.Stat(path); err == nil {
			Logger.WithFields(logrus.Fields{
				"reqpath":  r.URL.Path,
				"oldsize":  stat.Size(),
				"newsize":  fh.Size,
				"filename": fh.Filename,
			}).Warn("Overwriting file")
		}

		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, cm.mode)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, f)
		return err
	}

	for i := 1; i <= len(fhs); i++ {
		fh := fhs[i-1]
		Logger.Debugf("Processing file [%d of %d] %q", i, len(fhs), fh.Filename)
		if err := processFile(fh); err != nil {
			render.Render(w, r, ErrInternal(
				errors.WithMessage(err, fmt.Sprintf("file #%d, name=%q", i, fh.Filename))))
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

func (cm FSContentManager) DeleteFile(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(cm.pathBuilder(r), chi.URLParam(r, "name"))

	if err := os.Remove(path); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var ScriptMgr = FSContentManager{
	maxSize: 100 << 20, // Up to 100MB files
	mode:    0755,
	pathBuilder: func(*http.Request) string {
		return appCfg.ServiceMonitor.ChecksDir
	},
}

func RunScriptTest(w http.ResponseWriter, r *http.Request) {
	dir := appCfg.ServiceMonitor.ChecksDir
	path := filepath.Join(dir, chi.URLParam(r, "name"))
	abspath, err := filepath.Abs(path)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	args := []string{}
	if err := render.DecodeJSON(r.Body, &args); err != nil {
		if err == io.EOF {
			render.Render(w, r, ErrInvalidRequest(
				errors.New("request body must be array of script args (include '[]')")))
		} else {
			render.Render(w, r, ErrInvalidRequest(err))
		}
		return
	}

	buf := &bytes.Buffer{}
	cmd := exec.Command(abspath, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = buf, buf

	err = cmd.Start()
	if err != nil {
		render.Render(w, r, ErrInternal(errors.WithMessage(err, "unable to run command")))
		return
	}

	code, status := getCmdResult(cmd, appCfg.ServiceMonitor.Timeout)

	// NOTE(tbutts): If the command is killed by timeout, internal stdout/err buffers can't
	// be finalized properly by the os/exec package, leaving null bytes in our buffer.
	// This routine cleans them up.
	if status == models.ExitStatusTimeout {
		b := buf.Bytes()
		offset := bytes.IndexByte(b, '\x00')
		Logger.Debugf(`len=%d, offset to nul=%d`, buf.Len(), offset)
		buf.Truncate(offset)
	}

	msg := fmt.Sprintf("[%s] exit code: %v", status, code)
	buf.WriteString(msg + "\n")

	// send stdout + stderr from the command to the user
	render.PlainText(w, r, buf.String())

	// also log the script run on the server
	log := Logger.WithFields(logrus.Fields{"script": path, "args": args, "status": msg})
	if code > 0 {
		log.Warn("script test run failed")
	} else {
		log.Info("script test run passed")
	}
}

// Scoring analytics & graphs (admin-only)

func GetBreakdownOfSubmissionsPerFlag(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerFlag(db)
	ApiQuery(w, r, brkdwn, err)
}

func GetEachTeamsCapturedFlags(w http.ResponseWriter, r *http.Request) {
	brkdwn, err := models.ChallengeCapturesPerTeam(db)
	ApiQuery(w, r, brkdwn, err)
}
