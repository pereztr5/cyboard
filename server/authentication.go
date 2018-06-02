package server

import (
	"crypto/rand"
	"encoding/gob"
	"net/http"
	"time"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/cookiestore"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

const (
	FormCredsTeam = "teamname"
	FormCredsPass = "password"
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

func getSigningKey() []byte {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		panic(err)
	}
	return key[:]
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
