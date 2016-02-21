package cmd

import (
	"log"
	"net/http"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2/bson"
)

// Code provided by https://github.com/elithrar
func CheckSessionID(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		log.Printf("Getting from Store failed: %v", err)
		http.Error(w, http.StatusText(400), 400)
		return
	}
	// Get the ID from the session (if it exists) and
	// look up the associated team
	id, ok := session.Values["id"]
	if ok {
		t, err := GetTeamById(id.(*bson.ObjectId))
		if err != nil {
			// context.Set(r, "team", nil)
			// HTTP 500 here: GetTeamByID returning an error is probably a DB error?
			log.Printf("GetTeamById %v: %v", id, err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// It's valid! Set it in the context to look it up later.
		context.Set(r, "team", t)
	}
	next(w, r)
}

func RequireLogin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	t := context.Get(r, "team")
	if t != nil {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login.html", 302)
	}
}
