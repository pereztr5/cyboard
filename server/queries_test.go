package server

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var TestTeams = []Team{
	{bson.NewObjectId(), "blueteam", 1, "team1", "127.0.0.1", genPass("pass1"), ""},
	{bson.NewObjectId(), "blueteam", 2, "team2", "127.0.0.2", genPass("pass2"), ""},
}

var ScoreCategories = []string{"CTF", "Service"}

func init() {
	SetupScoringLoggers(&LogSettings{Level: "warn", Stdout: true})
	ensureTestDB()
}

func ensureTestDB() {
	SetupMongo(&DBSettings{URI: "mongodb://127.0.0.1:27017", DBName: "test"}, []string{})

	if DBSession() == nil {
		os.Exit(1)
	}
}

func cleanupDB() {
	for _, c := range []*mgo.Collection{Teams(), Challenges(), DB().C("results")} {
		c.RemoveAll(nil)
	}
}

func genPass(s string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Unable to create password from %q: %v", s, err)
		os.Exit(1)
	}
	return string(hash)
}

func TestDataGetTeams(t *testing.T) {
	cleanupDB()
	var docs []interface{}
	for _, doc := range TestTeams {
		docs = append(docs, doc)
	}
	Teams().Insert(docs...)

	for _, tt := range TestTeams {
		if tt.Group == "blueteam" {
			continue
		}

		var ok bool
		for _, dbTeam := range DataGetTeams() {
			if tt.Name == dbTeam.Name && tt.Number == dbTeam.Number {
				ok = true
			}
		}
		assert.True(t, ok, "Failed to retrieve blue team: %+v", tt)
	}
}

func TestDataAddTeams(t *testing.T) {
	cleanupDB()
	assert.Nil(t, DataAddTeams(TestTeams))

	dbTeams := []Team{}
	Teams().Find(nil).All(&dbTeams)
	for _, tt := range TestTeams {
		assert.Contains(t, dbTeams, tt, "Database did not contain the expected test team %+v", tt)
	}
}

func createTestScoreResults(teams []Team, times int) []Result {
	results := make([]Result, 0, len(ScoreCategories)*len(teams))
	for idx, team := range teams {
		for _, cat := range ScoreCategories {
			res := Result{
				Type:       cat,
				Teamname:   team.Name,
				Teamnumber: team.Number,
				Details:    "Just testing",
				Points:     idx + 1,
			}
			// Add the result a few times, just to confirm the group-by aggregation works
			for i := 0; i < times; i++ {
				results = append(results, res)
			}
		}
	}
	return results
}

func setupForDataGetAllScore(t *testing.T, times int) {
	cleanupDB()
	assert.Nil(t, DataAddTeams(TestTeams))

	results := createTestScoreResults(TestTeams, times)
	for _, r := range results {
		assert.Nil(t, DataAddResult(r, false))
	}
}

func TestDataGetAllScore(t *testing.T) {
	times := 3
	setupForDataGetAllScore(t, times)
	scores := DataGetAllScore()

	for _, score := range scores {
		for idx, team := range TestTeams {
			if score.Teamnumber == team.Number {
				assert.Equal(t, (idx+1)*times*2, score.Points, "Category '%v' did not match for team '%v'", score.Type, score.Teamname)
			}
		}
	}
}

func TestDataGetAllScoreSplitByType(t *testing.T) {
	times := 5
	setupForDataGetAllScore(t, times)
	scores := DataGetAllScoreSplitByType()

	satisfies := map[int]map[string]bool{}
	for _, team := range TestTeams {
		satisfies[team.Number] = map[string]bool{}
		for _, cat := range ScoreCategories {
			satisfies[team.Number][cat] = false
		}
	}

	for _, score := range scores {
		for idx, team := range TestTeams {
			if score.Teamnumber == team.Number {
				ok := assert.Equal(t, (idx+1)*times, score.Points, "Category '%v' did not match for team '%v'", score.Type, score.Teamname)
				satisfies[team.Number][score.Type] = ok
			}
		}
	}

	for team, matrix := range satisfies {
		for cat, ok := range matrix {
			assert.True(t, ok, "Team%d category '%v' was not satisfied.", team, cat)
		}
	}
}
