package cmd

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	gob.Register(&Team{})
	gob.Register(new(bson.ObjectId))
}

var Store *sessions.CookieStore

func CreateStore(hashkey, blockkey string) {
	hk, err := hex.DecodeString(hashkey)
	if err != nil {
		log.Fatalf("Could not decode hashkey: %v", err)
	}
	bk, err := hex.DecodeString(blockkey)
	if err != nil {
		log.Fatalf("Could not decode blockkey: %v", err)
	}
	Store = sessions.NewCookieStore([]byte(hk), []byte(bk))
}

func CheckCreds(w http.ResponseWriter, r *http.Request) (bool, error) {
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
		// Don't set the team to nil when authentication fails.
		// context.Set(r, "team", nil)
		return false, errors.New("Invalid Creds")
	}

	context.Set(r, "team", t)
	session.Values["id"] = t.Id
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return false, err
	}

	return true, nil
}
