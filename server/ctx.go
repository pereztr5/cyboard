package server

import (
	"net/http"

	"github.com/pereztr5/cyboard/server/models"
)

type CtxKey int8

const (
	ctxTeam = CtxKey(iota)
	ctxOwnedChallenges
)

func getCtxTeam(r *http.Request) models.Team {
	return r.Context().Value("team").(models.Team)
	// return r.Context().Value(ctxTeam).(*models.Team)
}

func getCtxOwnedChallenges(r *http.Request) []models.Challenge {
	return r.Context().Value(ctxOwnedChallenges).([]models.Challenge)
}
