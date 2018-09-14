package server

import (
	"context"
	"net/http"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/sirupsen/logrus"
)

type CtxKey int8

const (
	ctxTeam = CtxKey(iota)
	ctxOwnedChallenges
	ctxErrorMsgFields

	ctxUrlParamPrefix = "cyboard.param."
)

func getCtxTeam(r *http.Request) *models.Team {
	if t := r.Context().Value(ctxTeam); t != nil {
		return t.(*models.Team)
	}
	return nil
}

func saveCtxTeam(r *http.Request, team *models.Team) context.Context {
	return context.WithValue(r.Context(), ctxTeam, team)
}

func getCtxOwnedChallenges(r *http.Request) []models.Challenge {
	return r.Context().Value(ctxOwnedChallenges).([]models.Challenge)
}

type M map[string]interface{}

func saveCtxErrMsgFields(r *http.Request, fields M) context.Context {
	errMsgFields := getCtxErrMsgFields(r)
	for k, v := range fields {
		errMsgFields[k] = v
	}
	return context.WithValue(r.Context(), ctxErrorMsgFields, logrus.Fields(errMsgFields))
}

func getCtxErrMsgFields(r *http.Request) logrus.Fields {
	var fields logrus.Fields
	fields, ok := r.Context().Value(ctxErrorMsgFields).(logrus.Fields)
	if !ok {
		fields = make(logrus.Fields)
	}
	return fields
}

func saveCtxUrlParam(r *http.Request, name string, v interface{}) context.Context {
	return context.WithValue(r.Context(), ctxUrlParamPrefix+name, v)
}

func getCtxUrlParamInt(r *http.Request, name string) int {
	return r.Context().Value(ctxUrlParamPrefix + name).(int)
}

func getCtxIdParam(r *http.Request) int {
	return getCtxUrlParamInt(r, "id")
}
