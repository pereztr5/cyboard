package cmd

import (
	"fmt"

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
	Value       string        `json:"value"`
	Hints       []string      `json:"hints"`
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
		fmt.Printf("Error finding team %s err: %v\n", teamname, err)
	}
	return t, err
}

func GetTeamById(id bson.ObjectId) (Team, error) {
	var err error
	t := Team{}

	session, teamCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err = teamCollection.Find(bson.M{"_id": id}).One(&t)
	if err != nil {
		fmt.Printf("Error finding team %s err: %v\n", t.Teamname, err)
	}
	return t, err
}

// Query statements
func DataGetFlags() ([]Flag, error) {
	result := []Flag{}

	session, flagCollection := GetSessionAndCollection("flags")
	defer session.Close()

	err := flagCollection.Find(nil).Select(bson.M{"_id": 0, "value": 0}).All(&result)
	if err != nil {
		fmt.Printf("Could not get flags\n")
	}
	return result, nil
}

func DataCheckFlag(chal, val string) (bool, error) {
	var found bool
	result := Flag{}

	session, flagCollection := GetSessionAndCollection("flags")
	defer session.Close()

	err := flagCollection.Find(bson.M{"challenge": chal, "value": val}).Select(bson.M{"name": 1, "points": 1}).One(&result)
	if err != nil {
		// Log invalid flags here by team
		fmt.Printf("Invalid flag\n")
	} else {
		// Need to add points to the team who got the flag
		found = true
	}
	return found, nil
}

/*
func DataGetTeamScore() ([]Team, err) {
}
*/
