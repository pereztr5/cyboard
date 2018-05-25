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
func GetTeamByTeamname(teamname string) (models.Team, error) {
	t := models.Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"name": teamname}).One(&t)
	if err != nil {
		Logger.Printf("Error finding team by Teamname %s err: %v\n", teamname, err)
		return t, err
	}
	return t, nil
}

func GetTeamById(id *bson.ObjectId) (models.Team, error) {
	t := models.Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"_id": id}).One(&t)
	if err != nil {
		Logger.Printf("Error finding team by ID %v err: %v\n", id, err)
		return t, err
	}
	return t, nil
}

// Get Team name and ip only used for service checking
func DataGetTeamIps() ([]models.Team, error) {
	t := []models.Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"group": "blueteam"}).Select(bson.M{"_id": false, "name": true, "number": true, "ip": true}).All(&t)
	if err != nil {
		//Logger.Printf("Error finding teams: %v\n", err)
		return t, err
	}
	return t, nil
}

func DataGetTeams() []models.Team {
	t := []models.Team{}

	session, chalCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := chalCollection.Find(bson.M{"group": "blueteam"}).Sort("number").Select(bson.M{"_id": 0, "number": 1, "name": 1}).All(&t)
	if err != nil {
		Logger.Error("Could not get team info: ", err)
		return t
	}
	return t
}

// Gets everything about all users. Utilized by Admin dashboard
func DataGetAllUsers() []models.Team {
	var t []models.Team

	session, chalCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := chalCollection.Find(nil).
		Sort("group", "number").
		Select(bson.M{"_id": 0}).
		All(&t)
	if err != nil {
		Logger.Error("Failed to retrieve all users: ", err)
		return t
	}
	return t
}

// Query statements
func DataGetChallenges(groups []string) ([]models.Challenge, error) {
	challenges := []models.Challenge{}

	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()

	// TODO: .Select to include fields required, rather than excluding unsafe ones
	err := chalCollection.
		Find(bson.M{"group": bson.M{"$in": groups}}).
		Sort("description").
		Select(bson.M{"_id": 0, "flag": 0}).
		All(&challenges)
	if err != nil {
		return challenges, err
	}
	return challenges, nil
}

// FlagState represents the possibilities when user submits a flag guess
type FlagState int

const (
	// ValidFlag is for successful guesses of flags which were not previously submitted
	ValidFlag FlagState = 0
	// InvalidFlag is for bad guesses
	InvalidFlag = 1
	// AlreadyCaptured is for flags that were claimed by the team already
	AlreadyCaptured = 2
)

func DataCheckFlag(team models.Team, chal models.Challenge) (FlagState, error) {
	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()

	query := bson.M{"flag": chal.Flag}
	if len(chal.Name) > 0 {
		query["name"] = chal.Name
	}
	err := chalCollection.Find(query).One(&chal)
	if err != nil {
		if err == mgo.ErrNotFound {
			CaptFlagsLogger.WithField("team", team.Name).WithField("guess", chal.Flag).WithField("challenge", chal.Name).Println("Bad guess")
			return InvalidFlag, nil
		}
		return InvalidFlag, err
	}
	if HasFlag(team.Number, chal.Group, chal.Name) {
		// Got challenge already
		return AlreadyCaptured, nil
	}

	result := models.Result{
		Type:       "CTF",
		Timestamp:  time.Now(),
		Group:      chal.Group,
		Teamname:   team.Name,
		Teamnumber: team.Number,
		Details:    chal.Name,
		Points:     chal.Points,
	}
	CaptFlagsLogger.WithField("team", result.Teamname).WithField("challenge", result.Details).WithField("chalGroup", result.Group).
		WithField("points", result.Points).Println("Score!!")
	test := false
	return ValidFlag, DataAddResult(result, test)
}

func HasFlag(teamnumber int, challengeGroup, challengeName string) bool {
	session, resultCollection := GetSessionAndCollection("results")
	defer session.Close()

	cnt, err := resultCollection.
		Find(bson.M{"type": CTF, "group": challengeGroup, "teamnumber": teamnumber, "details": challengeName}).
		Count()
	if err != nil {
		Logger.WithError(err).Errorf("HasFlag failed for team '%d' for challenge '%s'", teamnumber, challengeName)
		return true
	}
	return cnt > 0
}

type ChallengeCount struct {
	Group  string `json:"group" bson:"_id"`
	Amount int    `json:"count"`
}

// DataGetTotalChallenges returns all the CTF groups and counts the amount
// of challenges in each group. E.g.
// `[{"group": "Wireless", "amount": 4}, {"group": "Embedded", "amount": 3}]`
func DataGetTotalChallenges() ([]ChallengeCount, error) {
	session, collection := GetSessionAndCollection("challenges")
	defer session.Close()
	totals := []ChallengeCount{}
	err := collection.Pipe([]bson.M{
		{"$group": bson.M{"_id": "$group", "amount": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"_id": 1}},
	}).All(&totals)
	if err != nil {
		Logger.Error("Error getting challenges: ", err)
		return totals, err
	}
	return totals, nil
}

// DataGetTeamChallenges returns the amount of each CTF group that has been
// scored by a given team. E.g. "team1" may have results such as
// `[{"group": "Wireless", "amount" 4}, {"group": "Embedded", "amount": 0}]`
func DataGetTeamChallenges(teamname string) ([]ChallengeCount, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	acquired := []ChallengeCount{}
	challengeGroups := DataGetChallengeGroupsList()
	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"type": CTF, "group": bson.M{"$in": challengeGroups}, "teamname": teamname}},
		{"$group": bson.M{"_id": "$group", "amount": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"_id": 1}},
	}).All(&acquired)
	if err != nil {
		Logger.Error("Error getting challenges: ", err)
		return acquired, err
	}

	if len(challengeGroups) != len(acquired) {
		for idx, chalGroup := range challengeGroups {
			if idx >= len(acquired) {
				acquired = append(acquired, ChallengeCount{Group: chalGroup})
			} else if acquired[idx].Group != chalGroup {
				acquired = append(acquired, ChallengeCount{})
				copy(acquired[idx+1:], acquired[idx:])
				acquired[idx] = ChallengeCount{Group: chalGroup}
			}
		}
	}

	return acquired, nil
}

// Find all the challenges in a group.
// If flagGroup param is empty "", all challenges are returned.
func DataGetChallengesByGroup(flagGroup string) ([]models.Challenge, error) {
	session, collection := GetSessionAndCollection("challenges")
	defer session.Close()

	filter := bson.M{}
	if flagGroup != "" {
		filter["group"] = flagGroup
	}

	var chals []models.Challenge
	return chals, collection.Find(filter).Sort("group", "name").All(&chals)
}

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
	var cResults []ServiceStatus

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
	var challenges []string
	err := collection.Find(nil).Distinct("group", &challenges)
	if err != nil {
		Logger.WithError(err).Error("Failed to query distinct Challenge groups")
	}
	return challenges
}

func DataGetServiceList() []string {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	var list []string

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
	var docs []interface{}
	for _, team := range teams {
		docs = append(docs, interface{}(team))
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

func DataDeleteTeam(teamName string) error {
	session, teamC := GetSessionAndCollection("teams")
	defer session.Close()
	return teamC.Remove(bson.M{"name": teamName})
}

func DataGetChallengeByName(name string) (chal models.Challenge, err error) {
	session, collection := GetChallengesCollection()
	defer session.Close()
	return chal, collection.Find(bson.M{"name": name}).One(&chal)
}

func DataAddChallenge(chal *models.Challenge) error {
	session, collection := GetChallengesCollection()
	defer session.Close()
	return collection.Insert(&chal)
}

func DataDeleteChallenge(id *bson.ObjectId) error {
	session, collection := GetChallengesCollection()
	defer session.Close()
	return collection.RemoveId(&id)
}

func DataUpdateChallenge(id *bson.ObjectId, updateOp *models.Challenge) error {
	session, collection := GetChallengesCollection()
	defer session.Close()
	return collection.UpdateId(id, &updateOp)
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

	var aggrResult []FlagSubmissions
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

	var aggrResult []CapturedFlagsOfTeam
	return aggrResult, collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "CTF", "group": bson.M{"$in": challengeGroups}}},
		{"$sort": bson.M{"details": 1}},
		{"$group": bson.M{"_id": "$teamname", "flags": bson.M{"$push": "$details"}}},
		{"$sort": bson.M{"_id": 1}},
	}).All(&aggrResult)
}
