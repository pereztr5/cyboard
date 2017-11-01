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
		Logger.Error("Getting session cookie from Store failed: ", err)
	}
	ctx := context.WithValue(r.Context(), "session", session)
	var team interface{} = nil
	if id, ok := session.Values["id"]; ok {
		t, err := GetTeamById(id.(*bson.ObjectId))
		if err != nil {
			Logger.Errorf("GetTeamById %v: %v", id, err)
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

func RequireGroupIsAnyOf(whitelistedGroups []string) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		team := r.Context().Value("team").(Team)
		for _, group := range whitelistedGroups {
			if team.Group == group {
				next(w, r)
				return
			}
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}

func RequireAdmin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	team := r.Context().Value("team").(Team)
	if team.Group == "admin" {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login", 302)
	}
}

func RequireCtfGroupOwner(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	t := r.Context().Value("team").(Team)
	if !AllowedToConfigureChallenges(t) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	chals, err := DataGetChallengesByGroup(t.AdminOf)
	if err != nil {
		Logger.Error("RequireCtfGroupOwner: failed to get flags by group: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	ctx := context.WithValue(r.Context(), ctxOwnedChallenges, chals)
	next(w, r.WithContext(ctx))
}

func AllowedToConfigureChallenges(t Team) bool {
	switch t.Group {
	case "admin", "blackteam":
		return true
	case "blueteam":
		return false
	default:
		return t.AdminOf != ""
	}
}
