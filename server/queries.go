package server

import (
	"fmt"

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
