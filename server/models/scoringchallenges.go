package models

import "github.com/sirupsen/logrus"

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

	if len(chal.Name) > 0 {
		sqlwhere = `c.flag = $2 AND c.Name = $3`
	} else {
		sqlwhere = `c.flag = $2`
	}

	// Use left outer joins to get the challenge and possible releated team if
	// the flag has been scored before. Examines the ctf_solve table to look for
	// whether the submitted flag was scored before. If the flag guess is entirely
	// incorrect, no row gets returned. If the team scored before, a full row
	// with the team's id is returned. If the flag is correct and not scored
	// by the team, the row returned won't have the team's id.
	sqlstr = `select c.id, c.name, c.category, c.Total, t.id ` +
		`from cyboard.challenge AS c ` +
		`LEFT JOIN cyboard.ctf_solve as solve ` +
		`ON c.id = solve.challenge_id AND ` + sqlwhere + ` ` +
		`LEFT JOIN cyboard.team as t ` +
		`ON solve.team_id = t.id AND t.id = $1 ` +
		`WHERE ` + sqlwhere

	if len(chal.Name) > 0 {
		err = db.QueryRow(sqlstr, team.ID, chal.Flag, chal.Name).Scan(challengeID, &chal.Name, &chal.Category, &points, solverID)
	} else {
		err = db.QueryRow(sqlstr, team.ID, chal.Flag).Scan(challengeID, &chal.Name, &chal.Category, &points, solverID)
	}

	if err != nil {
		if err == pgx.ErrNotFound {
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
	if err = db.Exec(sqlinsert, team.ID, chal.ID); err != nil {
		return InvalidFlag, err
	}

	CaptFlagsLogger.WithFields(logrus.Fields{"team": team.Name, "challenge": chal.Name, "category": chal.Category, "points": points}).Println("Score!!")
	return ValidFlag, nil
}
