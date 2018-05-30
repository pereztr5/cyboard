package server

import (
	"encoding/gob"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pereztr5/cyboard/server/models"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

const (
	FormCredsTeam = "teamname"
	FormCredsPass = "password"
)

func init() {
	gob.Register(&models.Team{})
	gob.Register(new(bson.ObjectId))
}

var Store *sessions.CookieStore

func CreateStore() {
	Store = sessions.NewCookieStore([]byte(securecookie.GenerateRandomKey(64)), []byte(securecookie.GenerateRandomKey(32)))
	Store.Options = &sessions.Options{
		// Cookie will last for 1 hour
		MaxAge:   3600,
		HttpOnly: true,
	}
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

	session, err := Store.Get(r, "cyboard")
	if err != nil {
		Logger.Error("Error getting session: ", err)
		return false
	}

	session.Values["id"] = t.Id
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		Logger.Error("Error saving session: ", err)
		return false
	}
	return true
}
