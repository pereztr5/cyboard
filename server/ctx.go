package server

import (
	"net/http"
)

type CtxKey int8

const (
	ctxTeam = CtxKey(iota)
	ctxOwnedChallenges
)

func getCtxTeam(r *http.Request) Team {
	return r.Context().Value("team").(Team)
	// return r.Context().Value(ctxTeam).(*Team)
}

func getCtxOwnedChallenges(r *http.Request) []Challenge {
	return r.Context().Value(ctxOwnedChallenges).([]Challenge)
}
