package server

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// setupResponder scaffolds some additional error logging that will occur whenever
// one of the go-chi/render methods are provided an `error` type to respond to the
// user with at the end of an http request.
func setupResponder(logger *logrus.Logger) {
	/* This function adapted from:
	https://github.com/go-chi/chi/blob/0c5e7abb4e562fa14dd2548cb57b28f979a7dcd9/_examples/rest/main.go#L261 */

	render.Respond = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		err, ok := v.(*ErrResponse)
		if ok && err.Err != nil {
			// We set a default error status response code if one hasn't been set.
			if _, ok := r.Context().Value(render.StatusCtxKey).(int); !ok {
				w.WriteHeader(http.StatusBadRequest)
			}

			logmsg := logger.WithError(err.Err).WithField("path", r.URL.Path)
			if fields, ok := r.Context().Value(ctxErrorMsgFields).(logrus.Fields); ok {
				logmsg = logmsg.WithFields(fields)
			}
			logmsg.Error("error during request")

			team := getCtxTeam(r)
			if team != nil {
				switch team.RoleName {
				// Only expose the error to staff
				case models.TeamRoleAdmin, models.TeamRoleCtfCreator:
					err.ErrorText = err.Err.Error()
				}
			}
			render.DefaultResponder(w, r, err)
			return
		}

		render.DefaultResponder(w, r, v)
	}
}

// RenderQueryErr logs a SQL related error and displays an appropriate message
// to the user. Additional logging context may be added by setting fields in
// the request context, see `getCtxErrMsgFields()`
func RenderQueryErr(w http.ResponseWriter, r *http.Request, err error) {
	switch errors.Cause(err) {
	case pgx.ErrNoRows:
		render.Render(w, r, ErrNotFound)
	default:
		render.Render(w, r, ErrInternal(errors.WithMessage(err, "GET "+r.URL.Path)))
	}
}

// ErrResponse renderer type for handling all sorts of errors.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

// Render satisfies the go-chi/render.Renderer interface, making errors easy to
// reuse and print out as JSON/XML
func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// ErrInvalidBecause is a public error where the reason text is always displayed,
// even to non-staff members.
// Handlers for login and flag submission will give more friendly errors with this.
// However, things like Internal Server Errors should never be exposed to non-staff.
func ErrInvalidBecause(reason string) render.Renderer {
	return &ErrResponse{HTTPStatusCode: 400, StatusText: reason}
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err: err, HTTPStatusCode: 400, StatusText: "Invalid request",
	}
}

func ErrRendering(err error) render.Renderer {
	return &ErrResponse{
		Err: err, HTTPStatusCode: 422, StatusText: "Error rendering response",
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrResponse{
		Err: err, HTTPStatusCode: 500, StatusText: "Internal server error",
	}
}

var (
	ErrNotFound  = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found"}
	ErrForbidden = &ErrResponse{HTTPStatusCode: 403, StatusText: "Forbidden"}
)
