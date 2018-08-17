package server

import (
	"html/template"
	"strings"

	"github.com/pereztr5/cyboard/server/models"
)

func buildHelperMap() template.FuncMap {
	return template.FuncMap{
		// Generic string methods
		"title":       strings.Title,
		"StringsJoin": strings.Join,

		"isChallengeOwner": allowedToConfigureChallenges,
	}
}

func allowedToConfigureChallenges(t *models.Team) bool {
	switch t.RoleName {
	case models.TeamRoleAdmin, models.TeamRoleCtfCreator:
		return true
	default:
		return false
	}
}
