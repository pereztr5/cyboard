package server

import (
	"crypto/rand"
	"encoding/gob"
	"net/http"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/cookiestore"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

const (
	FormCredsTeam = "teamname"
	FormCredsPass = "password"

	sessionConfigCollection = "session.config"
)

func init() {
	gob.Register(new(bson.ObjectId))
}

var sessionManager *scs.Manager

func CreateStore() {
	key := getSigningKey()
	sessionManager = scs.NewManager(cookiestore.New(key))
	sessionManager.Name("cyboard")
	sessionManager.Lifetime(1 * time.Hour)
	sessionManager.Persist(true)
	sessionManager.Secure(true)
	sessionManager.HttpOnly(true)
}

func CheckCreds(w http.ResponseWriter, r *http.Request) bool {
	teamname, password := r.FormValue(FormCredsTeam), r.FormValue(FormCredsPass)

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
	err = session.PutObject(w, "id", &t.Id)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		Logger.Error("Error saving session: ", err)
	}
	return err == nil
}

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
