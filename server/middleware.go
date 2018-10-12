package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
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

func RequireEventStarted(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isCtfStaff(getCtxTeam(r)) || time.Now().After(appCfg.Event.Start) {
			next.ServeHTTP(w, r)
			return
		}

		// TODO: This is my first try at selecting responses based on browser req vs ajax.
		// You could look at the "Accept" http header, but it can often have */* wildcards from
		// both the browser and cli tools, so that doesn't identify reliably. Look into further.
		if strings.HasPrefix(r.URL.Path, "/api") {
			render.Render(w, r, ErrForbiddenBecause("The competition has not started yet"))
		} else {
			http.Redirect(w, r, "/", 301)
		}
	})
}

func RequireEventNotOver(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isCtfStaff(getCtxTeam(r)) || time.Now().Before(appCfg.Event.End) {
			next.ServeHTTP(w, r)
			return
		}
		render.Render(w, r, ErrForbiddenBecause("The competition is over, go home"))
	})
}

func RequireUrlParamInt(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := strconv.Atoi(chi.URLParam(r, name))
			if err != nil {
				render.Render(w, r, ErrInvalidRequest(
					errors.WithMessage(err, fmt.Sprintf("%q URL Param", name))))
				return
			}
			ctx := saveCtxUrlParam(r, name, p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var RequireIdParam = RequireUrlParamInt("id")

func RequireNoSpeeding(lmt *limiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				http.Error(w, "Context was canceled", http.StatusServiceUnavailable)
				return
			default:
				httpError := tollbooth.LimitByRequest(lmt, w, r)
				if httpError != nil {
					w.Header().Add("Content-Type", lmt.GetMessageContentType())
					w.WriteHeader(httpError.StatusCode)
					w.Write([]byte(httpError.Message))
					return
				}

				next.ServeHTTP(w, r)
			}
		})
	}
}

func NewRateLimiter(rate float64) *limiter.Limiter {
	opts := &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}
	lmt := tollbooth.NewLimiter(rate, opts).
		SetMessage("Slow down there, tiger")
	return lmt
}

func MaybeRateLimit(r chi.Router, ratePerSec float64) chi.Router {
	if appCfg.Server.RateLimit {
		return r.With(RequireNoSpeeding(NewRateLimiter(ratePerSec)))
	}
	return r
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
