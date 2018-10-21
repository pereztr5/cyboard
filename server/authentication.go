package server

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/cookiestore"
	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	formCredsTeam = "teamname"
	formCredsPass = "password"

	sessionIDKey       = "id"
	sessionPGConfigKey = "session.config"
)

var sessionManager *scs.Manager

// CreateStore initializes the global Session Manager, used to authenticate
// users across requests. If secure is true, the generated browser cookies
// will only be shared over HTTPS.
func CreateStore(secure bool) {
	key := getSigningKey()
	sessionManager = scs.NewManager(cookiestore.New(key))
	sessionManager.Name("cyboard")
	sessionManager.Lifetime(7 * 24 * time.Hour)
	sessionManager.Persist(true)
	sessionManager.Secure(secure)
	sessionManager.HttpOnly(true)
	sessionManager.SameSite("Lax")
}

// CheckCreds authenticates users based on username/password form values contained
// in the request. If the credentials all match, the team's ID will be saved to a
// cookie in the browser. If there are any errors, they will get logged and this
// will return false.
func CheckCreds(w http.ResponseWriter, r *http.Request) bool {
	teamname, password := r.FormValue(formCredsTeam), r.FormValue(formCredsPass)

	t, err := models.TeamByName(db, teamname)
	if err != nil {
		return false
	}

	if err = bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(password)); err != nil {
		if err != bcrypt.ErrMismatchedHashAndPassword {
			Logger.Error(err)
		}
		return false
	}

	session := sessionManager.Load(r)
	err = session.PutInt(w, sessionIDKey, t.ID)
	if err != nil {
		Logger.Error("Error saving session: ", err)
	}
	return err == nil
}

// CheckSessionID is a middleware that authenticates users based on a cookie
// their browser supplies with each request that has their team's ID. This does
// a query against the database and sticks the matching models.Team values onto
// the request context, under the "team" key.
//
// If the user hasn't logged in, has tampered with their cookie, or there's
// some internal server error, the "team" key in the context will be nil.
func CheckSessionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var team *models.Team

		session := sessionManager.Load(r)
		hasID, err := session.Exists(sessionIDKey)
		if err != nil {
			Logger.WithError(err).Error("CheckSessionID: failed to load session data")
		} else if hasID {
			teamID, err := session.GetInt(sessionIDKey)
			if err != nil {
				Logger.WithError(err).Errorf("CheckSessionID: failed to load %q", sessionIDKey)
			} else {
				t, err := models.TeamByID(db, teamID)
				if err != nil {
					Logger.WithError(err).WithField("teamID", teamID).
						Error("CheckSessionID: GetTeamById failed")
				} else {
					team = t
				}
			}
		}
		ctx := saveCtxTeam(r, team)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getSigningKey fetches the session signing secret from the database, or
// creates a new one if one doesn't exist and saves that to the database.
func getSigningKey() []byte {
	// The signing key is stored in the db config table as a base64 string, and then decoded to its bytes
	var (
		s string
		b []byte
	)

	err := db.QueryRow(`SELECT value FROM cyboard.config WHERE key = $1`, sessionPGConfigKey).Scan(&s)
	if err != nil {
		if err != pgx.ErrNoRows {
			Logger.WithError(err).Fatal("getSigningKey: failed to fetch from postgres")
		}

		Logger.Info("Generating new session signing key")

		b = make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			Logger.WithError(err).Fatal("getSigningKey: failed to generate session signing key")
		}

		s = base64.StdEncoding.EncodeToString(b)

		_, err = db.Exec(`INSERT INTO cyboard.config (key, value) VALUES ($1,$2)`, sessionPGConfigKey, s)
		if err != nil {
			Logger.WithError(err).Fatal("getSigningKey: failed to insert new key into postgres")
		}
	}
	b, err = base64.StdEncoding.DecodeString(s)
	if err != nil {
		Logger.WithError(err).Fatal("getSigningKey: failed to decode key from postgres")
	}

	return b
}
