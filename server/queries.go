package server

import "gopkg.in/mgo.v2/bson"

type Team struct {
	Id     bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Number int           `json:"number"`
	Name   string        `json:"name"`
	Ip     string        `json:"ip"`
	Hash   string        `json:"-"`
}

type Challenge struct {
	Id          bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Group       string        `json:"group"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Flag        string        `json:"-" bson:"-"`
	Points      int           `json:"points"`
}

type Result struct {
	Id         bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Type       string        `json:"type" bson:"type"`
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

	err := teamCollection.Find(nil).Select(bson.M{"_id": false, "name": true, "number": true, "ip": true}).All(&t)
	if err != nil {
		//Logger.Printf("Error finding teams: %v\n", err)
		return t, err
	}
	return t, nil
}

// Query statements
func DataGetChallenges(group string) ([]Challenge, error) {
	challenges := []Challenge{}

	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()

	err := chalCollection.Find(bson.M{"group": group}).Sort("description").Select(bson.M{"_id": 0, "flag": 0}).All(&challenges)
	if err != nil {
		return challenges, err
	}
	return challenges, nil
}

func DataCheckAllFlags(team Team, flag string) (int, error) {
	chal := Challenge{}

	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()

	err := chalCollection.Find(bson.M{"flag": flag}).Select(bson.M{"_id": 0, "flag": 0}).One(&chal)
	if err != nil {
		// Wrong flag = 1
		return 1, err
	} else {
		if !HasFlag(team.Name, chal.Name) {
			// Correct flag = 0
			result := Result{
				Type:       "CTF",
				Group:      chal.Group,
				Teamname:   team.Name,
				Teamnumber: team.Number,
				Details:    chal.Name,
				Points:     chal.Points,
			}
			return 0, DataAddResult(result)
		} else {
			// Got challenge already
			return 2, nil
		}
	}
}

func DataCheckFlag(team Team, challengeName, flag string) (int, error) {
	chal := Challenge{}

	session, chalCollection := GetSessionAndCollection("challenges")
	defer session.Close()

	err := chalCollection.Find(bson.M{"flag": flag, "name": challengeName}).Select(bson.M{"_id": 0, "flag": 0}).One(&chal)
	if err != nil {
		// Wrong flag = 1
		return 1, err
	} else {
		if !HasFlag(team.Name, chal.Name) {
			// Correct flag = 0
			result := Result{
				Type:       "CTF",
				Group:      chal.Group,
				Teamname:   team.Name,
				Teamnumber: team.Number,
				Details:    chal.Name,
				Points:     chal.Points,
			}
			return 0, DataAddResult(result)
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
		Logger.Printf("Error getting challenges: %v\n", err)
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
		Logger.Printf("Error getting challenges: %v\n", err)
		return acquired, err
	}
	return acquired, nil
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
		Logger.Printf("Error getting team points: %v\n", err)
	} else {
		p = points["points"].(int)
	}
	return p
}

func DataGetAllScore() []Result {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	tmScore := []Result{}

	pipe := collection.Pipe([]bson.M{
		{"$group": bson.M{"_id": bson.M{"tname": "$teamname", "tnum": "$teamnumber"}, "points": bson.M{"$sum": "$points"}}},
		{"$project": bson.M{"_id": 0, "points": 1, "teamnumber": "$_id.tnum", "teamname": "$_id.tname"}},
		{"$sort": bson.M{"teamnumber": 1}},
	})
	err := pipe.All(&tmScore)
	if err != nil {
		Logger.Printf("Error getting all team scores: %v\n", err)
	}
	return tmScore
}

func DataGetServiceStatus() []Result {
	r := []Result{}

	session, results := GetSessionAndCollection("results")
	defer session.Close()

	err := results.Find(bson.M{"type": "Service"}).All(&r)
	if err != nil {
		Logger.Printf("Error getting service status: %v\n", err)
	}
	return r
}

// Insert statements
func DataAddResult(result Result) error {
	session, collection := GetSessionAndCollection("results")
	defer session.Close()
	err := collection.Insert(result)
	if err != nil {
		//Logger.Printf("Error inserting %s to team %s: %v", result.Details, result.Teamname, err)
		return err
	}
	return nil
}
