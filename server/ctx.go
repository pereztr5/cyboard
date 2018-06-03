package server

import (
	"context"
	"net/http"

	"github.com/pereztr5/cyboard/server/models"
)

type CtxKey int8

const (
	ctxTeam = CtxKey(iota)
	ctxOwnedChallenges
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
