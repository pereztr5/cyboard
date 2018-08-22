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
	}
}

func isAdmin(t *models.Team) bool {
	return t.RoleName == models.TeamRoleAdmin
}

func isCtfStaff(t *models.Team) bool {
	switch t.RoleName {
	case models.TeamRoleAdmin, models.TeamRoleCtfCreator:
		return true
	default:
		return false
	}
}
