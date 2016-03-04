package cmd

import (
	"gopkg.in/mgo.v2/bson"
)

type Team struct {
	Id         bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Teamname   string        `json:"teamname"`
	Hash       string        `json:"-"`
	Teamnumber int           `json:"teamnumber"`
	Services   []Service     `json:"services"`
	Flags      []Flag        `json:"flags"`
	Checks     []Check       `json:"checks"`
}

type Flag struct {
	Id          bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Flagname    string        `json:"flagname"`
	Challenge   string        `json:"challenge"`
	Points      int           `json:"points"`
	Description string        `json:"description"`
	Value       string        `json:"value" bson:"-"`
	Hints       []string      `json:"hints" bson:"-"`
}

type Service struct {
	Service string `json:"service"`
	Ip      string `json:"ip"`
}

type Check struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Points  int    `json:"points"`
}

// Authentication Queries
func GetTeamByTeamname(teamname string) (Team, error) {
	t := Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamCollection.Find(bson.M{"teamname": teamname}).Select(bson.M{"_id": 1, "teamnumber": 1, "teamname": 1, "hash": 1}).One(&t)
	if err != nil {
		Logger.Printf("Error finding team by Teamname %s err: %v\n", teamname, err)
		return t, err
	}
	return t, nil
}

func GetTeamById(id *bson.ObjectId) (Team, error) {
	var err error
	t := Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err = teamCollection.Find(bson.M{"_id": id}).One(&t)
	if err != nil {
		Logger.Printf("Error finding team by ID %v err: %v\n", id, err)
		return t, err
	}
	return t, nil
}

// Query statements
func DataGetFlags() ([]Flag, error) {
	result := []Flag{}

	session, flagCollection := GetSessionAndCollection("flags")
	defer session.Close()

	err := flagCollection.Find(nil).Select(bson.M{"_id": 0, "value": 0}).All(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func DataCheckFlag(teamname, chal, val string) (int, error) {
	result := Flag{}

	session, flagCollection := GetSessionAndCollection("flags")
	defer session.Close()

	err := flagCollection.Find(bson.M{"challenge": chal, "value": val}).Select(bson.M{"_id": 0, "value": 0, "hints": 0}).One(&result)
	if err != nil {
		// Wrong flag = 1
		return 1, err
	} else {
		// Correct flag = 0
		return 0, DataAddFlag(teamname, result)
	}
}

/*
func DataGetTeamScore() ([]Team, err) {
}
*/

// Insert statements
func DataAddFlag(teamname string, flag Flag) error {
	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()
	err := teamCollection.Update(
		bson.M{"teamname": teamname},
		bson.M{"$push": bson.M{"flags": flag}},
	)
	if err != nil {
		Logger.Printf("Error inserting flag %s to team %s: %v", flag.Flagname, teamname, err)
		return err
	}
	return nil
}
