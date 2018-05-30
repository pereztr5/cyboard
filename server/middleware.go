package server

import (
	"context"
	"net/http"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/urfave/negroni"

	"gopkg.in/mgo.v2/bson"
)

// Code provided by https://github.com/elithrar
func CheckSessionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value("team") != nil {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/login", 302)
		}
	})
}

type RequireGroupIsAnyOf struct {
	whitelistedGroups []string
}

func (mw RequireGroupIsAnyOf) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		team := r.Context().Value("team").(models.Team)
		for _, group := range mw.whitelistedGroups {
			if team.Group == group {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		team := r.Context().Value("team").(models.Team)
		if team.Group == "admin" {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/login", 302)
		}
	})
}

func RequireCtfGroupOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := r.Context().Value("team").(models.Team)
		if !allowedToConfigureChallenges(t) {
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
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UnwrapNegroniMiddleware(nh negroni.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nh.ServeHTTP(w, r, next.ServeHTTP)
		})
	}
}

func NegroniResponseWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(negroni.NewResponseWriter(w), r)
	})
}
