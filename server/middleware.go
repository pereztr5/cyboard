package server

import (
	"net/http"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2/bson"
)

// Code provided by https://github.com/elithrar
func CheckSessionID(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		Logger.Printf("Getting from Store failed: %v", err)
	}
	context.Set(r, "session", session)
	if id, ok := session.Values["id"]; ok {
		t, err := GetTeamById(id.(*bson.ObjectId))
		if err != nil {
			Logger.Printf("GetTeamById %v: %v", id, err)
			context.Set(r, "team", nil)
		} else {
			context.Set(r, "team", t)
		}
	} else {
		context.Set(r, "team", nil)
	}
	next(w, r)
}

func RequireLogin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if context.Get(r, "team") != nil {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}
