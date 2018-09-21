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

		"isAdmin":    isAdmin,
		"isCtfStaff": isCtfStaff,
		"isBlueteam": isBlueteam,
	}
}

func isAdmin(t *models.Team) bool {
	return t != nil && t.RoleName == models.TeamRoleAdmin
}

func isCtfStaff(t *models.Team) bool {
	if t == nil {
		return false
	}

	switch t.RoleName {
	case models.TeamRoleAdmin, models.TeamRoleCtfCreator:
		return true
	default:
		return false
	}
}

func isBlueteam(t *models.Team) bool {
	return t != nil && t.RoleName == models.TeamRoleBlueteam
}
