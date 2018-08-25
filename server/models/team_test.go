package models

import (
	"testing"

	"github.com/jackc/pgx"
	"github.com/stretchr/testify/assert"
)

func Test_TeamByID(t *testing.T) {
	prepareTestDatabase(t)
	tests := map[string]struct {
		team_id int
		err     error
	}{
		"known id returns correct team": {1, nil},
		"unknown id will error":         {-1, pgx.ErrNoRows},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			team, err := TeamByID(db, tt.team_id)
			if assert.Equal(t, tt.err, err) && tt.err == nil {
				assert.Equal(t, tt.team_id, team.ID)
			}
		})
	}
}

func Test_AllBlueteams(t *testing.T) {
	prepareTestDatabase(t)

	// team.yml creates 5 teams. 3 of them are blueteams, but 1 is deactivated.
	const blueteamCount = 2
	teams, err := AllBlueteams(db)
	if assert.Nil(t, err) {
		assert.Equal(t, blueteamCount, len(teams))
	}
}
