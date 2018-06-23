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

// Query statements

type ServiceStatus struct {
	Service      string `json:"service" bson:"_id"`
	TeamStatuses []struct {
		Name   string `json:"name"`
		Number int    `json:"number"`
		Status string `json:"status"`
	} `json:"teams" bson:"teams"`
}

// DataGetServiceStatus retrieves data for the Service Status page,
// which displays big pass/fail boxes for each teams' services
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
