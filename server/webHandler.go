package server

import (
	"context"
	"encoding/gob"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	gob.Register(&Team{})
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

func CheckCreds(w http.ResponseWriter, r *http.Request) (bool, *http.Request) {
	teamname, password := r.FormValue("teamname"), r.FormValue("password")
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		Logger.Error("Error getting session: ", err)
	}

	t, err := GetTeamByTeamname(teamname)
	if err != nil {
		return false, r
	}

	err = bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(password))
	if err != nil {
		Logger.Error("Invalid credentials: ", err)
		return false, r
	}

	r = r.WithContext(context.WithValue(r.Context(), "team", t))
	session.Values["id"] = t.Id
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		Logger.Error("Error saving session: ", err)
		return false, r
	}
	return true, r
}
