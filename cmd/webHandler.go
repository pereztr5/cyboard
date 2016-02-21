package cmd

import (
	"encoding/gob"
	"errors"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	_ "gopkg.in/mgo.v2/bson"
)

func init() {
	gob.Register(&Team{})
}

var Store = sessions.NewCookieStore(
	[]byte(securecookie.GenerateRandomKey(64)), //Signing key
	[]byte(securecookie.GenerateRandomKey(32)),
)

func CheckCreds(r *http.Request) (bool, error) {
	teamname, password := r.FormValue("teamname"), r.FormValue("password")
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		return false, err
	}
	t, err := GetTeamByTeamname(teamname)
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(t.Hash), []byte(password))
	if err != nil {
		context.Set(r, "team", nil)
		return false, errors.New("Invalid Creds")
	}
	context.Set(r, "team", t)
	session.Values["id"] = t.Id
	return true, nil
}
