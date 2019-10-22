package models

import (
	"context"
	"time"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

// CtfSolve represents a row from 'cyboard.ctf_solve'.
type CtfSolve struct {
	CreatedAt   time.Time `json:"created_at"`   // created_at
	TeamID      int       `json:"team_id"`      // team_id
	ChallengeID int       `json:"challenge_id"` // challenge_id
}

// Insert a scored flag into the database. Congrats!
func (cs *CtfSolve) Insert(db DB) error {
	const sqlstr = `INSERT INTO ctf_solve (team_id, challenge_id) VALUES ($1, $2)`
	_, err := db.Exec(sqlstr, cs.TeamID, cs.ChallengeID)
	return err
}

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
// all hidden flags. The Category ane Name are filled in on a successful guess.
type ChallengeGuess struct {
	Name     string `json:"name"`     // name
	Category string `json:"category"` // category
	Flag     string `json:"flag"`     // flag
}

// CheckFlagSubmission will award the team with a captured flag if their flag string
// guess is correct. No points will be given on a repeat flag, or obviously if the
// flag submitted is simply wrong.
func CheckFlagSubmission(db TXer, ctx context.Context, team *Team, chal *ChallengeGuess) (FlagState, error) {
	var (
		err         error
		challengeID int
		solverID    *int
		points      float32

		sqlwhere string
		sqlstr   string
	)
	tx, err := db.BeginEx(ctx, &pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return InvalidFlag, err
	}
	defer tx.Rollback()

	// Examines the ctf_solve table to look for whether the submitted flag was scored before.
	// If the flag guess is entirely incorrect, no row gets returned.
	// If the team scored before, a full row with the team's id is returned.
	// If the flag is correct and not scored by the team, the row returned will have a null team id.
	sqlstr = `SELECT c.id, c.name, c.category, c.total, solve.team_id
	FROM challenge AS c
	LEFT JOIN ctf_solve solve ON c.id = solve.challenge_id AND solve.team_id = $1
	WHERE `

	// If the flag guess is for a specific flag, only check if that one is correct.
	// Otherwise, check if any hidden/anonymous flags have the guessed string value.
	if len(chal.Name) > 0 {
		sqlwhere = `c.hidden = false AND c.flag = $2 AND c.Name = $3`
		err = tx.QueryRow(sqlstr+sqlwhere, team.ID, chal.Flag, chal.Name).Scan(&challengeID, &chal.Name, &chal.Category, &points, &solverID)
	} else {
		sqlwhere = `c.hidden = true AND c.flag = $2`
		err = tx.QueryRow(sqlstr+sqlwhere, team.ID, chal.Flag).Scan(&challengeID, &chal.Name, &chal.Category, &points, &solverID)
	}

	if err != nil {
		return InvalidFlag, err
	} else if solverID != nil {
		// Got challenge already
		return AlreadyCaptured, nil
	}

	award := CtfSolve{ChallengeID: challengeID, TeamID: team.ID}
	if err = award.Insert(tx); err != nil {
		return InvalidFlag, err
	}

	if err = tx.Commit(); err != nil {
		return InvalidFlag, errors.WithMessage(err, "CheckFlagSubmission: failed to commit transaction")
	}
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
		`LEFT JOIN ctf_solve AS solve ON solve.challenge_id = id AND solve.team_id = $1 ` +
		`WHERE challenge.hidden = false ` +
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

// ChallengeCaptureCount holds the number of teams that have beaten a CTF challenge,
// and the first team w/ timestamp to solve it.
type ChallengeCaptureCount struct {
	Designer  string     `json:"designer"`             // challenge.designer
	Category  string     `json:"category"`             // challenge.category
	Name      string     `json:"name"`                 // challenge.name
	Count     int        `json:"count"`                // --- (calculated column)
	FirstTeam *string    `json:"first_team,omitempty"` // team.name
	Timestamp *time.Time `json:"timestamp,omitempty"`  // ctf_solve.created_at
}

// ChallengeCapturesPerFlag gets the number of times each flag was captured,
// sorted by designer, then category, then name. This includes flags that remain
// unsolved, which will have a Count of 0, and team/timestamp of nil.
func ChallengeCapturesPerFlag(db DB) ([]ChallengeCaptureCount, error) {
	// NOTE: This query is pretty unoptimized, with misuse of timescale's `first` aggregate,
	// which is a sequential scan.
	// But the volume of data (a few hundred rows?) is so low it doesn't matter.
	const sqlstr = `SELECT designer, category, c.name, COUNT(t.id),
		first(t.name, solve.created_at), first(solve.created_at, solve.created_at)
	FROM challenge AS c
	LEFT JOIN ctf_solve AS solve ON solve.challenge_id = id
	LEFT JOIN team AS t ON solve.team_id = t.id
	GROUP BY c.name, category, designer
	ORDER BY designer, category, name`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ccs := []ChallengeCaptureCount{}
	for rows.Next() {
		cc := ChallengeCaptureCount{}
		if err = rows.Scan(&cc.Designer, &cc.Category, &cc.Name, &cc.Count,
			&cc.FirstTeam, &cc.Timestamp); err != nil {
			return nil, err
		}
		ccs = append(ccs, cc)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ccs, nil
}

// CapturedChallenge contains enough to identify a solved challenge.
type CapturedChallenge struct {
	Designer  string    `json:"designer"` // challenge.designer
	Category  string    `json:"category"` // challenge.category
	Name      string    `json:"name"`     // challenge.name
	Timestamp time.Time `json:"time"`     // ctf_solve.created_at
}

// TeamCapturedChallenges holds the flags a team has captured.
type TeamCapturedChallenges struct {
	Team       string              `json:"team"` // team.name
	Challenges []CapturedChallenge `json:"challenges"`
}

// ChallengeCapturesPerTeam retrieves each team with the flags they've captured.
func ChallengeCapturesPerTeam(db DB) ([]TeamCapturedChallenges, error) {
	const sqlstr = `SELECT team.name, ch.designer, ch.category, ch.name, cs.created_at
	FROM team
	  JOIN ctf_solve AS cs ON team.id = cs.team_id
	  JOIN challenge AS ch ON cs.challenge_id = ch.id
	ORDER BY team.id, ch.designer, ch.category, ch.name`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tccs := []TeamCapturedChallenges{}

	// Have to do the groupby team's name in golang, because with Go's type system
	// it is much more ceremony (and not type-safe) to decode the nested JSON object.
	var tname string
	for rows.Next() {
		cc := CapturedChallenge{}
		if err = rows.Scan(&tname, &cc.Designer, &cc.Category, &cc.Name, &cc.Timestamp); err != nil {
			return nil, err
		}

		// Postgres guarantees everything is sorted neatly, so just keep appending
		// to the team's solves list until the team name doesn't match up.
		if len(tccs) == 0 || tccs[len(tccs)-1].Team != tname {
			tmp := &TeamCapturedChallenges{}
			tmp.Team = tname
			tccs = append(tccs, *tmp)
		}
		teamSolves := &tccs[len(tccs)-1]
		teamSolves.Challenges = append(teamSolves.Challenges, cc)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tccs, nil
}

// CtfSolveResult represents a solved challenge, associated with the solving team,
// via the pivot table 'ctf_solve'
type CtfSolveResult struct {
	Timestamp     time.Time `json:"timestamp"`      // ctf_solve.created_at
	TeamID        int       `json:"team_id"`        // team.id
	TeamName      string    `json:"team_name"`      // team.name
	ChallengeID   int       `json:"challenge_id"`   // challenge.id
	ChallengeName string    `json:"challenge_name"` // challenge.name
	Category      string    `json:"category"`       // challenge.category
	Points        float32   `json:"points"`         // challenge.total
}

// ChallengeCapturesByTime fetches all `ctf_solve` rows with their ctf name/info and solving
// team's name & id, and orders them by time. A `cutoffTime` threshold will exclude
// any solves older than the date.
func ChallengeCapturesByTime(db DB, cutoffTime time.Time) ([]CtfSolveResult, error) {
	const sqlstr = `SELECT cs.created_at, t.id, t.name, c.id, c.name, c.category, c.total
	FROM ctf_solve cs
		JOIN team t ON cs.team_id = t.id
		JOIN challenge c ON c.id = cs.challenge_id
	WHERE cs.created_at > $1
	ORDER BY cs.created_at DESC`

	rows, err := db.Query(sqlstr, cutoffTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []CtfSolveResult{}
	for rows.Next() {
		x := CtfSolveResult{}
		err = rows.Scan(&x.Timestamp, &x.TeamID, &x.TeamName,
			&x.ChallengeID, &x.ChallengeName, &x.Category, &x.Points)
		if err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}
