package server

import (
	"html/template"
	"math/rand"
	"path/filepath"
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
		"fmtDuration": fmtDuration,

		// App-specific helpers
		"isAdmin":    isAdmin,
		"isCtfStaff": isCtfStaff,
		"isBlueteam": isBlueteam,
	}
}

func fmtTimestamp(t time.Time) string {
	return t.Format(time.Stamp)
}

func fmtDuration(d time.Duration) string {
	return d.Truncate(time.Second).String()
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

var homepageVideos []string

func getHomepageVid() string {
	// Cache webm files on first run
	if len(homepageVideos) == 0 {
		files, _ := filepath.Glob("ui/static/assets/media/madhacks/*.webm")
		// uh oh, no videos!
		if len(files) == 0 {
			return ""
		}

		for _, f := range files {
			homepageVideos = append(homepageVideos, strings.TrimPrefix(f, "ui/static/"))
		}
	}
	return homepageVideos[rand.Intn(len(homepageVideos))]
}
