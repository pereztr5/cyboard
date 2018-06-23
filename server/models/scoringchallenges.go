package models

import (
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

// FlagState represents the possibilities when user submits a flag guess
type FlagState int

const (
	// ValidFlag is for successful guesses of flags which were not previously submitted
	ValidFlag FlagState = 0
	// InvalidFlag is for bad guesses
	InvalidFlag = 1
	// AlreadyCaptured is for flags that were claimed by the team already
	AlreadyCaptured = 2
)

// ChallengeGuess is a blueteam's attempt to captured a flag. Only the Flag field
// is required to be set. Leaving Name empty causes the guess to checked against
// all flags, which is obviously much easier for the blueteams. This was a conscious
// event design decision.
type ChallengeGuess struct {
	Name     string `json:"name"`     // name
	Category string `json:"category"` // category
	Flag     string `json:"flag"`     // flag
}

// CheckFlagSubmission will award the team with a captured flag if their flag string
// guess is correct. No points will be given on a repeat flag, or obviously if the
// flag submitted is simply wrong.
func CheckFlagSubmission(db DB, team *Team, chal *ChallengeGuess) (FlagState, error) {
	var (
		err         error
		challengeID *int
		solverID    *int
		points      float32

		sqlwhere string
		sqlstr   string
	)

	// Examines the ctf_solve table to look for whether the submitted flag was scored before.
	// If the flag guess is entirely incorrect, no row gets returned.
	// If the team scored before, a full row with the team's id is returned.
	// If the flag is correct and not scored by the team, the row returned will have a null team id.
	sqlstr = `SELECT c.id, c.name, c.category, c.total, solve.team_id
	FROM challenge AS c
	LEFT JOIN ctf_solve solve ON c.id = solve.challenge_id AND solve.team_id = $1
	WHERE`

	if len(chal.Name) > 0 {
		sqlwhere = `c.hidden = false AND c.flag = $2 AND c.Name = $3`
		err = db.QueryRow(sqlstr+sqlwhere, team.ID, chal.Flag, chal.Name).Scan(challengeID, &chal.Name, &chal.Category, &points, solverID)
	} else {
		sqlwhere = `c.hidden = false AND c.flag = $2`
		err = db.QueryRow(sqlstr+sqlwhere, team.ID, chal.Flag).Scan(challengeID, &chal.Name, &chal.Category, &points, solverID)
	}

	if err != nil {
		if err == pgx.ErrNoRows {
			CaptFlagsLogger.WithField("team", team.Name).WithField("guess", chal.Flag).WithField("challenge", chal.Name).Println("Bad guess")
			return InvalidFlag, nil
		}
		return InvalidFlag, err
	}
	if solverID != nil {
		// Got challenge already
		return AlreadyCaptured, nil
	}

	const sqlinsert = `INSERT INTO cyboard.ctf_solve (team_id, challenge_id) VALUES ($1, $2)`
	if _, err = db.Exec(sqlinsert, team.ID, challengeID); err != nil {
		return InvalidFlag, err
	}

	CaptFlagsLogger.WithFields(logrus.Fields{"team": team.Name, "challenge": chal.Name, "category": chal.Category, "points": points}).Println("Score!!")
	return ValidFlag, nil
}

// CTFProgress holds a team's status for a ctf category.
// Represents e.g. `completed 4 out of 5 challenges in the reversing category`
type CTFProgress struct {
	Category string `json:"category"`
	Amount   int    `json:"count"`
	Max      int    `json:"max"`
}

// GetTeamCTFProgress retrieves the given team's status in each ctf category,
// by counting the number of solved challenges out of the total amount of them.
func GetTeamCTFProgress(db DB, teamID int) ([]CTFProgress, error) {
	const sqlstr = `SELECT category, COUNT(solve.team_id) AS amount, COUNT(*) AS max ` +
		`FROM challenge ` +
		`LEFT JOIN ctf_solve AS solve ON solve.challenge_id = id AND solve.team_id = $1` +
		`WHERE hidden = false ` +
		`GROUP BY category`

	rows, err := db.Query(sqlstr, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ccs := []CTFProgress{}
	for rows.Next() {
		cc := CTFProgress{}
		if err = rows.Scan(&cc.Category, &cc.Amount, &cc.Max); err != nil {
			return nil, err
		}
		ccs = append(ccs, cc)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ccs, nil
}
