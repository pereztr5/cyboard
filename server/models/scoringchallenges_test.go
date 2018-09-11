package models

import (
	"context"
	"testing"

	"github.com/jackc/pgx"
	"github.com/stretchr/testify/assert"
)

/* For these tests, the DB will be loaded with 3 teams and 1 challenge.
Only one team (team1, id=1) has 'solved' the one challenge.
Team2 (id=2) has made no progress in the CTF.
Team3 (id=3) is a disabled team.
*/

func Test_CheckFlagSubmission(t *testing.T) {
	teams_already_capped := func() *Team { return &Team{ID: 1, Name: "team1"} }
	teams_not_capped := func() *Team { return &Team{ID: 2, Name: "team2"} }

	guess_named_valid := func() *ChallengeGuess {
		return &ChallengeGuess{Name: "Totally Rad Challenge", Flag: "flag{its_ok_tobe_rad_sometimes}"}
	}
	guess_named_bad := func() *ChallengeGuess {
		return &ChallengeGuess{Name: "Totally Rad Challenge", Flag: "flag{Im_just_guessing_because_Im_a_dumbdumb}"}
	}

	guess_anon_valid := func() *ChallengeGuess {
		return &ChallengeGuess{Flag: "flag{its_ok_tobe_rad_sometimes}"}
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
		"New capture":    {team: teams_not_capped(), cg: guess_named_valid(), fs: ValidFlag, err: nil},
		"Failed capture": {team: teams_not_capped(), cg: guess_named_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},
		"Already capped": {team: teams_already_capped(), cg: guess_named_valid(), fs: AlreadyCaptured, err: nil},
		"Failed+Already": {team: teams_already_capped(), cg: guess_named_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},

		"Anon new capture":    {team: teams_not_capped(), cg: guess_anon_valid(), fs: ValidFlag, err: nil},
		"Anon fail capture":   {team: teams_not_capped(), cg: guess_anon_bad(), fs: InvalidFlag, err: pgx.ErrNoRows},
		"Anon already capped": {team: teams_already_capped(), cg: guess_anon_valid(), fs: AlreadyCaptured, err: nil},
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
	expected := []ChallengeCaptureCount{{Designer: "test_master", Category: "RAD", Name: "Totally Rad Challenge", Count: 1}}

	challenge_captures, err := ChallengeCapturesPerFlag(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected, challenge_captures, "RAD Challenge should be captued by one team (team1)")
	}
}

func Test_ChallengeCapturesPerTeam(t *testing.T) {
	prepareTestDatabase(t)
	expected := []TeamChallengeCaptures{{Team: "team1", Designer: "test_master", Category: "RAD", Challenge: "Totally Rad Challenge"}}

	per_team_captures, err := ChallengeCapturesPerTeam(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected, per_team_captures, "team1 should have the Totally Rad Challenge. No other ctf captures should exist.")
	}
}
