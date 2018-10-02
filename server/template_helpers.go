package server

import (
	"html/template"
	"strings"
	"time"

	"github.com/pereztr5/cyboard/server/models"
)

func buildHelperMap() template.FuncMap {
	return template.FuncMap{
		// Generic helper methods
		"title":       strings.Title,
		"StringsJoin": strings.Join,
		"timestamp":   fmtTimestamp,

		// App-specific helpers
		"isAdmin":    isAdmin,
		"isCtfStaff": isCtfStaff,
		"isBlueteam": isBlueteam,
	}
}

func fmtTimestamp(t time.Time) string {
	return t.Format(time.Stamp)
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
