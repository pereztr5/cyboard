package models

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx"
	"github.com/pereztr5/cyboard/server/apptest"
	"github.com/stretchr/testify/assert"
)

/* For these tests, the DB will be loaded with 3 teams and 2 challenges.
Challenge 1 (id=1) is public/named/not hidden.
Challenge 2 (id=2) is anonymous/unnamed/hidden.
`team1, id=1` has solved challenge 1.
`team2, id=2` has solved challenge 2.
`team3, id=3` is a disabled team.
*/

var (
	team1, team2 string
	time1, time2 time.Time
)

func init() {
	team1, team2 = "team1", "team2"
	time1, time2 = apptest.MustParseTime("2018-07-29T09:00:00.000-04:00"), apptest.MustParseTime("2018-07-29T09:05:00.000-04:00")
}

func Test_CheckFlagSubmission(t *testing.T) {
	team_already_capped_named := func() *Team { return &Team{ID: 1, Name: "team1"} }
	team_not_capped_anon := team_already_capped_named

	team_not_capped_named := func() *Team { return &Team{ID: 2, Name: "team2"} }
	team_already_capped_anon := team_not_capped_named

	guess_named_valid := func() *ChallengeGuess {
		return &ChallengeGuess{Name: "Totally Rad Challenge", Flag: "flag{its_ok_tobe_rad_sometimes}"}
	}
	guess_named_bad := func() *ChallengeGuess {
		return &ChallengeGuess{Name: "Totally Rad Challenge", Flag: "flag{Im_just_guessing_because_Im_a_dumbdumb}"}
	}
	guess_named_anon_valid := func() *ChallengeGuess {
		return &ChallengeGuess{Flag: "flag{its_ok_tobe_rad_sometimes}"}
	}

	guess_anon_valid := func() *ChallengeGuess {
		return &ChallengeGuess{Flag: "how am I supposed to know that!"}
	}
	guess_anon_bad := func() *ChallengeGuess {
		return &ChallengeGuess{Flag: "flag{Im_just_guessing_because_Im_a_dumbdumb"}
	}

	cases := map[string]struct {
		team *Team
		cg   *ChallengeGuess
		fs   FlagState
		err  error
	}{
		"Named New capture":    {team: team_not_capped_named(), cg: guess_named_valid(), fs: ValidFlag, err: nil},
		"Named Failed capture": {team: team_not_capped_named(), cg: guess_named_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},
		"Named Already capped": {team: team_already_capped_named(), cg: guess_named_valid(), fs: AlreadyCaptured, err: nil},
		"Named Failed+Already": {team: team_already_capped_named(), cg: guess_named_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},

		"Can't cap named anonymously": {team: team_not_capped_named(), cg: guess_named_anon_valid(), fs: InvalidFlag, err: pgx.ErrNoRows},

		"Anon new capture":    {team: team_not_capped_anon(), cg: guess_anon_valid(), fs: ValidFlag, err: nil},
		"Anon fail capture":   {team: team_not_capped_anon(), cg: guess_anon_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},
		"Anon already capped": {team: team_already_capped_anon(), cg: guess_anon_valid(), fs: AlreadyCaptured, err: nil},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			prepareTestDatabase(t)

			flagState, err := CheckFlagSubmission(db, context.Background(), tt.team, tt.cg)
			if assert.Equal(t, tt.err, err, "Expected error/no error did not occur") {
				assert.Equal(t, tt.fs, flagState, "The guess did not work as expected")
			} else {
				t.Logf("%v", err)
			}
		})
	}
}

func Test_GetTeamCTFProgress(t *testing.T) {
	prepareTestDatabase(t)

	var ctf_prog []CTFProgress
	var err error

	ctf_prog, err = GetTeamCTFProgress(db, 1)
	if assert.Nil(t, err) {
		expected := []CTFProgress{{Category: "RAD", Amount: 1, Max: 1}}
		assert.Equal(t, expected, ctf_prog, "Team 1 did not have the right ctf progress")
	}

	ctf_prog, err = GetTeamCTFProgress(db, 2)
	if assert.Nil(t, err) {
		expected := []CTFProgress{{Category: "RAD", Amount: 0, Max: 1}}
		assert.Equal(t, expected, ctf_prog, "Team 2 did not have the right ctf progress")
	}
}

func Test_ChallengeCapturesPerFlag(t *testing.T) {
	prepareTestDatabase(t)

	expected := []ChallengeCaptureCount{
		{Designer: "test_master", Category: "RAD", Name: "No challenge here", Count: 1, FirstTeam: &team2, Timestamp: &time2},
		{Designer: "test_master", Category: "RAD", Name: "Totally Rad Challenge", Count: 1, FirstTeam: &team1, Timestamp: &time1},
	}

	challenge_captures, err := ChallengeCapturesPerFlag(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected, challenge_captures, "RAD Challenge should be captued by one team (team1)")
	}
}

func Test_ChallengeCapturesPerTeam(t *testing.T) {
	prepareTestDatabase(t)
	expected := []TeamCapturedChallenges{
		{Team: "team1", Challenges: []CapturedChallenge{{Designer: "test_master", Category: "RAD", Name: "Totally Rad Challenge", Timestamp: time1}}},
		{Team: "team2", Challenges: []CapturedChallenge{{Designer: "test_master", Category: "RAD", Name: "No challenge here", Timestamp: time2}}},
	}

	per_team_captures, err := ChallengeCapturesPerTeam(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected, per_team_captures,
			"team1 should have the 'Totally Rad Challenge'. "+
				"team2 should have 'No challenge here'.")
	}
}

func Test_GetChallengeCapturesByTime(t *testing.T) {
	prepareTestDatabase(t)
	expected := []CtfSolveResult{
		{Timestamp: time2, TeamID: 2, TeamName: "team2", ChallengeID: 2, Category: "RAD", ChallengeName: "No challenge here", Points: 8},
		{Timestamp: time1, TeamID: 1, TeamName: "team1", ChallengeID: 1, Category: "RAD", ChallengeName: "Totally Rad Challenge", Points: 5},
	}

	cutoffDate := time1.Add(-time.Second)
	capsByTime, err := ChallengeCapturesByTime(db, cutoffDate)
	if assert.Nil(t, err) {
		assert.Equal(t, expected, capsByTime,
			"two results should be available. One for each team, with team2's"+
				"appearing first, per descending sort order.")
	}
}
