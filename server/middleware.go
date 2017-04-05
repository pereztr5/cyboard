package server

import (
	"context"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

// Code provided by https://github.com/elithrar
func CheckSessionID(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		Logger.Printf("Getting from Store failed: %v", err)
	}
	ctx := context.WithValue(r.Context(), "session", session)
	var team interface{} = nil
	if id, ok := session.Values["id"]; ok {
		t, err := GetTeamById(id.(*bson.ObjectId))
		if err != nil {
			Logger.Printf("GetTeamById %v: %v", id, err)
		} else {
			team = t
		}
	}
	ctx = context.WithValue(ctx, "team", team)
	next(w, r.WithContext(ctx))
}

func RequireLogin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Context().Value("team") != nil {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}
