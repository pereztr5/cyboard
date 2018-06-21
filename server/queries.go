package server

import (
	"fmt"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/pereztr5/cyboard/server/models"
	"gopkg.in/mgo.v2/bson"
)

const (
	CTF     = "CTF"
	Service = "Service"
)

var ScoreCategories = []string{CTF, Service}

// Authentication Queries

// Query statements
// DataGetTeamScore used on blueteam dash to display just their score
func DataGetTeamScore(teamname string) int {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	points := bson.M{}
	var p int
	pipe := collection.Pipe([]bson.M{
		{"$match": bson.M{"teamname": teamname}},
		{"$group": bson.M{"_id": nil, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "points": 1}},
	})
	err := pipe.One(&points)
	if err != nil {
		Logger.Errorf("Error getting `%s` points: %v", teamname, err)
	} else {
		p = points["points"].(int)
	}
	return p
}

// ScoreResult represents the data returned from a query for a team's score.
// This is slimmed down from the `models.Result` model type, to reduce processing & bandwidth,
// and improve ease-of-use.
type ScoreResult struct {
	Teamnumber int    `json:"teamnumber"`
	Teamname   string `json:"teamname"`
	Type       string `json:"type,omitempty"`
	Points     int    `json:"points"`
}

// DataGetAllScore _unused_
func DataGetAllScore() []ScoreResult {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	tmScore := []ScoreResult{}

	teams := DataGetTeams()
	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"points": bson.M{"$ne": 0}}},
		{"$group": bson.M{"_id": bson.M{"tname": "$teamname", "tnum": "$teamnumber"}, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "teamnumber": "$_id.tnum", "teamname": "$_id.tname", "points": 1}},
		{"$sort": bson.M{"teamnumber": 1}},
	}).All(&tmScore)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}
	// Get defaults for teams that do not have a score
	if l := len(tmScore); l < len(teams) {
		if l == 0 {
			for _, t := range teams {
				tmScore = append(tmScore, ScoreResult{Teamname: t.Name, Teamnumber: t.Number, Points: 0})
			}
		} else {
			for i := 0; i < len(teams); i++ {
				tmScore = append(tmScore, ScoreResult{Teamname: teams[i].Name, Teamnumber: teams[i].Number, Points: 0})
			}
		}
	}
	return tmScore
}

func DataGetAllScoreSplitByType() []ScoreResult {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	teams := DataGetTeams()
	tmScore := make([]ScoreResult, 0, len(teams)*2)

	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"points": bson.M{"$ne": 0}}},
		{"$group": bson.M{"_id": bson.M{"tname": "$teamname", "tnum": "$teamnumber", "type": "$type"}, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "teamnumber": "$_id.tnum", "teamname": "$_id.tname", "type": "$_id.type", "points": 1}},
		{"$sort": bson.M{"teamnumber": 1, "type": 1}},
	}).All(&tmScore)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}

	// Fill in default score for each category for every team
	if len(tmScore) != len(teams)*2 {
		var team *models.Team
		var cat *string
		for i := 0; i < len(teams)*2; i++ {
			team, cat = &teams[i/2], &ScoreCategories[i%2]
			if i >= len(tmScore) {
				tmScore = append(tmScore, ScoreResult{Teamname: team.Name, Teamnumber: team.Number, Type: *cat})
			} else if tmScore[i].Teamnumber != team.Number || (tmScore[i].Teamnumber == team.Number && tmScore[i].Type != *cat) {
				tmScore = append(tmScore, ScoreResult{})
				copy(tmScore[i+1:], tmScore[i:])
				tmScore[i] = ScoreResult{Teamname: team.Name, Teamnumber: team.Number, Type: *cat}
			}
		}
	}

	return tmScore
}

type ServiceStatus struct {
	Service      string `json:"service" bson:"_id"`
	TeamStatuses []struct {
		Name   string `json:"name"`
		Number int    `json:"number"`
		Status string `json:"status"`
	} `json:"teams" bson:"teams"`
}

func DataGetServiceStatus() []ServiceStatus {
	session, results := GetSessionAndCollection("results")
	defer session.Close()
	cResults := []ServiceStatus{}

	err := results.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service"}},
		{"$group": bson.M{"_id": bson.M{"service": "$group", "tnum": "$teamnumber", "tname": "$teamname"}, "status": bson.M{"$last": "$details"}}},
		{"$group": bson.M{"_id": "$_id.service", "teams": bson.M{"$addToSet": bson.M{"number": "$_id.tnum", "name": "$_id.tname", "status": "$status"}}}},
	}).All(&cResults)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}
	return cResults
}

func DataGetServiceResult() []models.Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	sList := []models.Result{}

	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service"}},
		{"$group": bson.M{"_id": bson.M{"service": "$group", "tnum": "$teamnumber", "tname": "$teamname"}, "status": bson.M{"$last": "$details"}}},
		{"$group": bson.M{"_id": "$_id.service", "teams": bson.M{"$addToSet": bson.M{"number": "$_id.tnum", "name": "$_id.tname", "status": "$status"}}}},
		{"$unwind": "$teams"},
		{"$project": bson.M{"_id": 0, "group": "$_id", "teamnumber": "$teams.number", "teamname": "$teams.name", "details": "$teams.status"}},
		{"$sort": bson.M{"group": 1, "teamnumber": 1}},
	}).All(&sList)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}
	return sList
}

// ServiceResult represents the data returned from a query regarding a team's latest service check.
type ServiceResult struct {
	Teamnumber int    `json:"teamnumber"`
	Details    string `json:"details"`
}

func DataGetResultByService(blueteams []models.Team, service string) []ServiceResult {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	teamStatus := make([]ServiceResult, 0, len(blueteams))

	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service", "group": service}},
		{"$group": bson.M{"_id": bson.M{"service": "$group", "tnum": "$teamnumber"}, "status": bson.M{"$last": "$details"}}},
		{"$project": bson.M{"_id": 0, "teamnumber": "$_id.tnum", "details": "$status"}},
		{"$sort": bson.M{"teamnumber": 1}},
	}).All(&teamStatus)
	if err != nil {
		Logger.Error("Error getting team status by service: ", err)
	}

	// Fill in an empty result for any teams without a status for any service
	if len(blueteams) != len(teamStatus) {
		for idx, team := range blueteams {
			// Search for teamStatuses being too short, or holes (team1, team2, ___, team4)
			if idx >= len(teamStatus) {
				teamStatus = append(teamStatus, ServiceResult{Teamnumber: team.Number, Details: "N/A"})
			} else if teamStatus[idx].Teamnumber != team.Number {
				teamStatus = append(teamStatus, ServiceResult{})
				copy(teamStatus[idx+1:], teamStatus[idx:])
				teamStatus[idx] = ServiceResult{Teamnumber: team.Number, Details: "N/A"}
			}
		}
	}
	return teamStatus
}

func DataGetChallengeGroupsList() []string {
	session, collection := GetSessionAndCollection("challenges")
	defer session.Close()
	challenges := []string{}
	err := collection.Find(nil).Distinct("group", &challenges)
	if err != nil {
		Logger.WithError(err).Error("Failed to query distinct Challenge groups")
	}
	return challenges
}

func DataGetServiceList() []string {
	// NOTE: This function is only used by the `/api/services` endpoint, which itself is unused.
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	list := []string{}

	err := collection.Find(bson.M{"type": "Service"}).Distinct("group", &list)
	if err != nil {
		Logger.Error("Error getting service list: ", err)
	}
	return list
}

// TODO: Combine queries since this has repeating code
func DataGetLastServiceResult() time.Time {
	session, results := GetSessionAndCollection("results")
	defer session.Close()
	id := bson.M{}
	err := results.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service"}},
		{"$sort": bson.M{"_id": 1}},
		{"$group": bson.M{"_id": nil, "last": bson.M{"$last": "$_id"}}},
		{"$project": bson.M{"_id": 0, "last": 1}},
	}).One(&id)
	var latest time.Time
	if err != nil {
		if err != mgo.ErrNotFound {
			Logger.Error("Error getting last Service result: ", err)
		}
	} else {
		latest = id["last"].(bson.ObjectId).Time()
	}
	return latest
}

func DataGetLastResult() time.Time {
	session, results := GetSessionAndCollection("results")
	defer session.Close()
	id := bson.M{}

	// TODO: Update to use the "timestamp" field, instead of bson native _id
	err := results.Pipe([]bson.M{
		{"$sort": bson.M{"_id": 1}},
		{"$group": bson.M{"_id": nil, "last": bson.M{"$last": "$_id"}}},
		{"$project": bson.M{"_id": 0, "last": 1}},
	}).One(&id)
	var latest time.Time
	if err != nil {
		if err != mgo.ErrNotFound {
			Logger.Error("Error getting last document: ", err)
		}
	} else {
		latest = id["last"].(bson.ObjectId).Time()
	}
	return latest
}

// Insert statements
func DataAddResult(result models.Result, test bool) error {
	var col string
	if !test {
		col = "results"
	} else {
		col = "testResults"
	}
	session, collection := GetSessionAndCollection(col)
	defer session.Close()
	err := collection.Insert(result)
	if err != nil {
		//Logger.Printf("Error inserting %s to team %s: %v", result.Details, result.Teamname, err)
		return err
	}
	return nil
}

func DataAddResults(results []models.Result, test bool) error {
	docs := make([]interface{}, len(results))
	for i, result := range results {
		docs[i] = result
	}

	collName := "results"
	if test {
		collName = "testResults"
	}
	session, collection := GetSessionAndCollection(collName)
	defer session.Close()
	return collection.Insert(docs...)
}

func DataAddTeams(teams []models.Team) error {
	session, teamC := GetSessionAndCollection("teams")
	defer session.Close()
	docs := make([]interface{}, len(teams))
	for i, team := range teams {
		docs[i] = team
	}
	err := teamC.Insert(docs...)
	if err != nil {
		return fmt.Errorf("failed to add new teams: mongo DB error: %v", err)
	}
	return nil
}

func DataUpdateTeam(teamName string, updateOp map[string]interface{}) error {
	// sanitization may panic if someone is sending crafted, bad JSON to the api,
	// but this is a gated, admin only operation.
	sanitizedOp, err := sanitizeUpdateOp(updateOp)
	if err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}
	session, teamC := GetSessionAndCollection("teams")
	defer session.Close()
	return teamC.Update(bson.M{"name": teamName}, bson.M{"$set": sanitizedOp})

}

func DataAddChallenges(team *models.Team, challenges []models.Challenge) error {
	docs := make([]interface{}, len(challenges))
	for i, chal := range challenges {
		if !ctfIsAdminOf(team, &chal) {
			return fmt.Errorf("AddChallenges: user %s (adminOf=%s) unauthorized to add flags into group: %v",
				team.Name, team.AdminOf, chal.Group)
		}
		docs[i] = chal
	}

	session, collection := GetChallengesCollection()
	defer session.Close()
	return collection.Insert(docs...)
}

// Score breakdown methods

type FlagSubmissions struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Submissions int    `json:"submissions"`
}

// Gets the number of times each flag was captured in any of 'challengeGroups'.
// Json results: [{ "name": "FLAG-01", "group": "Wireless", "submissions": 5 }, ...]
func DataGetSubmissionsPerFlag(challengeGroups []string) ([]FlagSubmissions, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()

	aggrResult := []FlagSubmissions{}
	return aggrResult, collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "CTF", "group": bson.M{"$in": challengeGroups}}},
		{"$group": bson.M{"_id": bson.M{"name": "$details", "group": "$group"}, "submissions": bson.M{"$sum": 1}}},
		{"$project": bson.M{"name": "$_id.name", "group": "$_id.group", "submissions": 1, "_id": 0}},
		{"$sort": bson.M{"name": 1, "submissions": -1}},
	}).All(&aggrResult)
}

type CapturedFlagsOfTeam struct {
	Team  string   `json:"team" bson:"_id"`
	Flags []string `json:"flags"`
}

// Gets the flags each team has captured in any of 'challengeGroups'.
// Json results: [{ "team": "team1", "flags": ["Crypto-04", "Wifi-01"]}, ...]
func DataGetEachTeamsCapturedFlags(challengeGroups []string) ([]CapturedFlagsOfTeam, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()

	aggrResult := []CapturedFlagsOfTeam{}
	return aggrResult, collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "CTF", "group": bson.M{"$in": challengeGroups}}},
		{"$sort": bson.M{"details": 1}},
		{"$group": bson.M{"_id": "$teamname", "flags": bson.M{"$push": "$details"}}},
		{"$sort": bson.M{"_id": 1}},
	}).All(&aggrResult)
}
