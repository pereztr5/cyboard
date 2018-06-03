package server

import (
	"crypto/rand"
	"encoding/gob"
	"net/http"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/cookiestore"
	"github.com/pereztr5/cyboard/server/models"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

const (
	formCredsTeam = "teamname"
	formCredsPass = "password"

	sessionIDKey            = "id"
	sessionConfigCollection = "session.config"
)

func init() {
	gob.Register(new(bson.ObjectId))
}

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
}

// CheckCreds authenticates users based on username/password form values contained
// in the request. If the credentials all match, the team's ID will be saved to a
// cookie in the browser. If there are any errors, they will get logged and this
// will return false.
func CheckCreds(w http.ResponseWriter, r *http.Request) bool {
	teamname, password := r.FormValue(formCredsTeam), r.FormValue(formCredsPass)

	t, err := GetTeamByTeamname(teamname)
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
	err = session.PutObject(w, sessionIDKey, &t.Id)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
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
			teamID := new(bson.ObjectId)

			err := session.GetObject(sessionIDKey, teamID)
			if err != nil {
				Logger.WithError(err).Errorf("CheckSessionID: failed to load %q", sessionIDKey)
			} else {
				t, err := GetTeamById(teamID)
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
	type sessionKeyInMongo struct {
		ID []byte `bson:"_id"`
	}

	key := sessionKeyInMongo{}

	dbSession, coll := GetSessionAndCollection(sessionConfigCollection)
	defer dbSession.Close()

	err := coll.Find(nil).One(&key)

	if err != nil {
		if err != mgo.ErrNotFound {
			Logger.WithError(err).Fatal("getSigningKey: failed to fetch from mongo")
		}

		Logger.Info("Generating new session signing key")

		key.ID = make([]byte, 32)
		_, err := rand.Read(key.ID)
		if err != nil {
			Logger.WithError(err).Fatal("getSigningKey: failed to generate session signing key")
		}

		err = coll.Insert(key)
		if err != nil {
			Logger.WithError(err).Fatal("getSigningKey: failed to insert new key into mongo")
		}
	}

	return key.ID
}
