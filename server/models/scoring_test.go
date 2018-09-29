package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_TeamsScores(t *testing.T) {
	prepareTestDatabase(t)
	expected_scores := []TeamsScoresResponse{
		{TeamID: 1, Name: "team1", Score: 15, Service: 4, Ctf: 5, Other: 5},
		{TeamID: 2, Name: "team2", Score: 10, Service: 2, Ctf: 8, Other: 0},
	}

	scores, err := TeamsScores(db)
	if assert.Nil(t, err) {
		assert.Equal(t, 2, len(scores), "Number of scoring teams")

		for idx, expected := range expected_scores {
			actual := scores[idx]
			assert.Equal(t, expected, actual, "Scores do not match: (team=%v), (idx=%d)", expected.Name, idx)
		}
	}
}

func Test_LatestScoreChange(t *testing.T) {
	prepareTestDatabase(t)
	const time_str = "2018-07-29T09:15:00.000-04:00"
	expected_ts, _ := time.Parse(time.RFC3339, time_str)

	latest_ts, err := LatestScoreChange(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected_ts, latest_ts)
	}
}
