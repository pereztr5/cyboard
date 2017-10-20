package server

import (
	"fmt"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Team struct {
	Id      bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Group   string        `json:"group"`
	Number  int           `json:"number"`
	Name    string        `json:"name"`
	Ip      string        `json:"ip"`
	Hash    string        `json:"-"`
	AdminOf string        `json:"adminof"`
}

type Challenge struct {
	Id          bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Group       string        `json:"group"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Flag        string        `json:"flag" bson:"flag"`
	Points      int           `json:"points"`
}

type Result struct {
	Id         bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Type       string        `json:"type" bson:"type"`
	Timestamp  time.Time     `json:"timestamp" bson:"timestamp"`
	Group      string        `json:"group" bson:"group"`
	Teamname   string        `json:"teamname" bson:"teamname"`
	Teamnumber int           `json:"teamnumber" bson:"teamnumber"`
	Details    string        `json:"details" bson:"details"`
	Points     int           `json:"points" bson:"points"`
}

// Authentication Queries
func GetTeamByTeamname(teamname string) (Team, error) {
	t := Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"name": teamname}).One(&t)
	if err != nil {
		Logger.Printf("Error finding team by Teamname %s err: %v\n", teamname, err)
		return t, err
	}
	return t, nil
}

func GetTeamById(id *bson.ObjectId) (Team, error) {
	t := Team{}

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
func DataGetTeamIps() ([]Team, error) {
	t := []Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"group": "blueteam"}).Select(bson.M{"_id": false, "name": true, "number": true, "ip": true}).All(&t)
	if err != nil {
		//Logger.Printf("Error finding teams: %v\n", err)
		return t, err
	}
	return t, nil
}

func DataGetTeams() []Team {
	t := []Team{}

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
func DataGetAllUsers() []Team {
	var t []Team

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
func DataGetChallenges(groups []string) ([]Challenge, error) {
	challenges := []Challenge{}

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

func DataCheckFlag(team Team, chal Challenge) (int, error) {
	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()
	var err error

	if len(chal.Name) > 0 {
		err = chalCollection.Find(bson.M{"flag": chal.Flag, "name": chal.Name}).Select(bson.M{"_id": 0, "flag": 0}).One(&chal)
	} else {
		err = chalCollection.Find(bson.M{"flag": chal.Flag}).Select(bson.M{"_id": 0, "flag": 0}).One(&chal)
	}
	if err != nil {
		// Wrong flag = 1
		return 1, err
	} else {
		if !HasFlag(team.Name, chal.Name) {
			// Correct flag = 0
			result := Result{
				Type:       "CTF",
				Timestamp:  time.Now(),
				Group:      chal.Group,
				Teamname:   team.Name,
				Teamnumber: team.Number,
				Details:    chal.Name,
				Points:     chal.Points,
			}
			CaptFlagsLogger.Printf("Team [%d] just scored '%d points' for flag '%s'!",
				result.Teamnumber, result.Points, result.Details)
			test := false
			return 0, DataAddResult(result, test)
		} else {
			// Got challenge already
			return 2, nil
		}
	}
}

func HasFlag(teamname, challengeName string) bool {
	chal := Challenge{}

	session, resultCollection := GetSessionAndCollection("results")
	defer session.Close()

	// TODO: Do not need the returned document.
	// Need to find better way to check if exists
	err := resultCollection.Find(bson.M{"teamname": teamname, "details": challengeName}).One(&chal)
	if err != nil {
		// TODO: Log error
		return false
	}
	return true
}

func DataGetTotalChallenges() (map[string]int, error) {
	session, collection := GetSessionAndCollection("challenges")
	defer session.Close()
	totals := make(map[string]int)
	t := bson.M{}
	pipe := collection.Pipe([]bson.M{
		{"$group": bson.M{"_id": "$group", "total": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"_id": -1}},
	})
	iter := pipe.Iter()
	for iter.Next(&t) {
		totals[t["_id"].(string)] = t["total"].(int)
	}
	if err := iter.Close(); err != nil {
		Logger.Error("Error getting challenges: ", err)
		return totals, err
	}
	return totals, nil
}

func DataGetTeamChallenges(teamname string) (map[string]int, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	acquired := make(map[string]int)
	a := bson.M{}
	pipe := collection.Pipe([]bson.M{
		{"$match": bson.M{"teamname": teamname, "type": "CTF"}},
		{"$group": bson.M{"_id": "$group", "acquired": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"_id": -1}},
	})
	iter := pipe.Iter()
	for iter.Next(&a) {
		acquired[a["_id"].(string)] = a["acquired"].(int)
	}
	if err := iter.Close(); err != nil {
		Logger.Error("Error getting challenges: ", err)
		return acquired, err
	}
	return acquired, nil
}

// Find all the challenges in a group.
// If flagGroup param is empty "", all challenges are returned.
func DataGetChallengesByGroup(flagGroup string) ([]Challenge, error) {
	session, collection := GetSessionAndCollection("challenges")
	defer session.Close()

	filter := bson.M{}
	if flagGroup != "" {
		filter["group"] = flagGroup
	}

	var chals []Challenge
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

func DataGetAllScore() []Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	tmScore := []Result{}

	// todo: optimize this query for an index
	//       Mongo does not let $groupby operations use indexes,
	//       but this query is used very often.
	teams := DataGetTeams()
	pipe := collection.Pipe([]bson.M{
		{"$group": bson.M{"_id": bson.M{"tname": "$teamname", "tnum": "$teamnumber"}, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "points": 1, "teamnumber": "$_id.tnum", "teamname": "$_id.tname"}},
		{"$sort": bson.M{"teamnumber": 1}},
	})
	err := pipe.All(&tmScore)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}
	// Get defaults for teams that do not have a score
	if l := len(tmScore); l < len(teams) {
		if l == 0 {
			for _, t := range teams {
				tmScore = append(tmScore, Result{Teamname: t.Name, Teamnumber: t.Number, Points: 0})
			}
		} else {
			for i := 0; i < len(teams); i++ {
				tmScore = append(tmScore, Result{Teamname: teams[i].Name, Teamnumber: teams[i].Number, Points: 0})
			}
		}
	}
	return tmScore
}

func DataGetAllScoreSplitByType() []Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	tmScore := []Result{}

	teams := DataGetTeams()
	pipe := collection.Pipe([]bson.M{
		{"$group": bson.M{"_id": bson.M{"tname": "$teamname", "tnum": "$teamnumber", "type": "$type"}, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "points": 1, "teamnumber": "$_id.tnum", "teamname": "$_id.tname", "type": "$_id.type"}},
		{"$sort": bson.M{"teamnumber": 1}},
	})
	err := pipe.All(&tmScore)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}

	// TODO: Handle a team without any score in either category more gracefully than this
	if len(tmScore) < len(teams)*2 {
		tmScore = []Result{}
		for _, t := range teams {
			tmScore = append(tmScore, Result{Teamname: t.Name, Teamnumber: t.Number})
		}
	}
	return tmScore
}

func DataGetServiceStatus() []interface{} {
	session, results := GetSessionAndCollection("results")
	defer session.Close()
	var cResults []interface{}

	pipe := results.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service"}},
		{"$group": bson.M{"_id": bson.M{"service": "$group", "tnum": "$teamnumber", "tname": "$teamname"}, "status": bson.M{"$last": "$details"}}},
		{"$group": bson.M{"_id": "$_id.service", "teams": bson.M{"$addToSet": bson.M{"number": "$_id.tnum", "name": "$_id.tname", "status": "$status"}}}},
	})
	err := pipe.All(&cResults)
	if err != nil {
		Logger.Error("Error getting all team scores: ", err)
	}
	return cResults
}

func DataGetServiceResult() []Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	sList := []Result{}

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

func DataGetResultByService(service string) []Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	teamStatus := []Result{}

	err := collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service", "group": service}},
		{"$group": bson.M{"_id": bson.M{"service": "$group", "tnum": "$teamnumber", "tname": "$teamname"}, "status": bson.M{"$last": "$details"}}},
		{"$project": bson.M{"_id": 0, "group": "$_id.service", "teamnumber": "$_id.tnum", "teamname": "$_id.tname", "details": "$status"}},
		{"$sort": bson.M{"teamnumber": 1}},
	}).All(&teamStatus)
	if err != nil {
		Logger.Error("Error getting team status by service: ", err)
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
	res := bson.M{}

	pipe := collection.Pipe([]bson.M{
		{"$match": bson.M{"type": "Service"}},
		{"$group": bson.M{"_id": "$group"}},
		{"$project": bson.M{"_id": 0, "group": "$_id"}},
		{"$sort": bson.M{"group": 1}},
	})
	iter := pipe.Iter()
	for iter.Next(&res) {
		list = append(list, res["group"].(string))
	}
	if err := iter.Close(); err != nil {
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
		Logger.Error("Error getting last Service result: ", err)
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
		Logger.Error("Error getting last document: ", err)
	} else {
		latest = id["last"].(bson.ObjectId).Time()
	}
	return latest
}

// Insert statements
func DataAddResult(result Result, test bool) error {
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

func DataAddTeams(teams []Team) error {
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

// Score breakdown methods

// Gets the number of times each flag was captured in any of 'challengeGroups'.
// Json results: [{ "name": "FLAG-01", "group": "Wireless", "submissions": 5 }, ...]
func DataGetSubmissionsPerFlag(challengeGroups []string) ([]bson.M, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()

	var aggrResult []bson.M
	return aggrResult, collection.Pipe([]bson.M{
		{"$match": bson.M{"group": bson.M{"$in": challengeGroups}}},
		{"$group": bson.M{"_id": bson.M{"name": "$details", "group": "$group"}, "submissions": bson.M{"$sum": 1}}},
		{"$project": bson.M{"name": "$_id.name", "group": "$_id.group", "submissions": 1, "_id": 0}},
		{"$sort": bson.M{"name": 1, "submissions": -1}},
	}).All(&aggrResult)
}

// Gets the flags each team has captured in any of 'challengeGroups'.
// Json results: [{ "team": "team1", "flags": ["Crypto-04", "Wifi-01"]}, ...]
func DataGetEachTeamsCapturedFlags(challengeGroups []string) ([]bson.M, error) {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()

	var aggrResult []bson.M
	return aggrResult, collection.Pipe([]bson.M{
		{"$match": bson.M{"group": bson.M{"$in": challengeGroups}}},
		{"$sort": bson.M{"details": 1}},
		{"$group": bson.M{"_id": "$teamname", "flags": bson.M{"$push": "$details"}}},
		{"$sort": bson.M{"_id": 1}},
		{"$project": bson.M{"team": "$_id", "flags": "$flags", "_id": 0}},
	}).All(&aggrResult)
}
