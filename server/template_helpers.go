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

		// Template helpers. Mostly just wrappers around queries.go functions to handle errors.
		"totalChallenges":    getChallenges,
		"teamChallenges":     getTeamChallenges,
		"allTeamScores":      getAllTeamScores,
		"getOwnedChalGroups": getChallengesOwnerOf,
		"existsSpecialFlags": existsSpecialFlags,
		"isChallengeOwner":   allowedToConfigureChallenges,

		// Straight database queries
		"allBlueTeams":   DataGetTeams,
		"teamScore":      DataGetTeamScore,
		"getStatus":      DataGetResultByService,
		"serviceList":    DataGetServiceList,
		"challengesList": DataGetChallengeGroupsList,
		"allUsers":       DataGetAllUsers,
	}

}

func getChallenges() []ChallengeCount {
	totals, err := DataGetTotalChallenges()
	if err != nil {
		Logger.Error("Could not get challenges: ", err)
	}
	return totals
}

func getTeamChallenges(teamname string) []ChallengeCount {
	acquired, err := DataGetTeamChallenges(teamname)
	if err != nil {
		Logger.Error("Could not get team challenges: ", err)
	}
	return acquired
}

func getAllTeamScores() []map[string]interface{} {
	results := DataGetAllScoreSplitByType()
	scores := make([]map[string]interface{}, 0, len(results)/2)

	acc := make(map[string]map[string]interface{})
	for _, r := range results {
		score, ok := acc[r.Teamname]
		if ok {
			score[r.Type] = r.Points
			score["Points"] = score["CTF"].(int) + score["Service"].(int)
			scores = append(scores, score)
		} else {
			acc[r.Teamname] = map[string]interface{}{
				"Teamnumber": r.Teamnumber,
				"Teamname":   r.Teamname,
				r.Type:       r.Points,
			}
		}
	}

	return scores
}

func allowedToConfigureChallenges(t models.Team) bool {
	switch t.Group {
	case "admin", "blackteam":
		return true
	case "blueteam":
		return false
	default:
		return t.AdminOf != ""
	}
}

func existsSpecialFlags() bool {
	return len(specialChallenges) > 0
}
