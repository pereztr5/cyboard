package server

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/pereztr5/cyboard/server/models"
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

func RequireGroupIsAnyOf(whitelistedGroups []models.TeamRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			team := getCtxTeam(r)
			for _, group := range whitelistedGroups {
				if team.RoleName == group {
					next.ServeHTTP(w, r)
					return
				}
			}
			render.Render(w, r, ErrForbidden)
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		team := getCtxTeam(r)
		if team.RoleName == models.TeamRoleAdmin {
			next.ServeHTTP(w, r)
		} else {
			render.Render(w, r, ErrForbidden)
		}
	})
}

func RequireCtfStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		team := getCtxTeam(r)

		switch team.RoleName {
		case models.TeamRoleAdmin, models.TeamRoleCtfCreator:
			next.ServeHTTP(w, r)
		default:
			render.Render(w, r, ErrForbidden)
		}
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
