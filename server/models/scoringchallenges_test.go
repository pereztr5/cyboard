package models

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		"Failed capture": {team: teams_not_capped(), cg: guess_named_bad(), fs: InvalidFlag, err: nil},
		"Already capped": {team: teams_already_capped(), cg: guess_named_valid(), fs: AlreadyCaptured, err: nil},
		"Failed+Already": {team: teams_already_capped(), cg: guess_named_bad(), fs: InvalidFlag, err: nil},

		"Anon new capture":    {team: teams_not_capped(), cg: guess_anon_valid(), fs: ValidFlag, err: nil},
		"Anon fail capture":   {team: teams_not_capped(), cg: guess_anon_bad(), fs: InvalidFlag, err: nil},
		"Anon already capped": {team: teams_already_capped(), cg: guess_anon_valid(), fs: AlreadyCaptured, err: nil},
		"Anon failed+already": {team: teams_already_capped(), cg: guess_anon_bad(), fs: InvalidFlag, err: nil},
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
