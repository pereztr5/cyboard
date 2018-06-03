package server

import (
	"context"
	"net/http"

	"github.com/urfave/negroni"
)

func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if getCtxTeam(r) != nil {
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
		team := getCtxTeam(r)
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
		team := getCtxTeam(r)
		if team.Group == "admin" {
			next.ServeHTTP(w, r)
		} else {
			http.Redirect(w, r, "/login", 302)
		}
	})
}

func RequireCtfGroupOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := getCtxTeam(r)
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
