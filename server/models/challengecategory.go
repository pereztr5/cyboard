// Package models contains the types for schema 'cyboard'.
package models

// ChallengeCategory represents a row from 'cyboard.challenge_category'.
type ChallengeCategory struct {
	Name string `json:"name"`
}

// Insert inserts the ChallengeCategory to the database.
func (cc *ChallengeCategory) Insert(db DB) error {
	const sqlstr = `INSERT INTO cyboard.challenge_category (` +
		`name` +
		`) VALUES (` +
		`$1` +
		`)`

	return db.QueryRow(sqlstr, cc.Name).Scan(&cc.Name)
}

// Update updates the Challenge in the database.
func (cc *ChallengeCategory) Update(db DB, name string) error {
	const sqlstr = `UPDATE cyboard.challenge_category SET (` +
		`name` +
		`) = ( ` +
		`$2` +
		`) WHERE id = $1`

	_, err := db.Exec(sqlstr, cc.Name, name)
	return err
}

// Delete deletes the ChallengeCategory from the database.
func (cc *ChallengeCategory) Delete(db DB) error {
	const sqlstr = `DELETE FROM cyboard.challenge_category WHERE name = $1`

	_, err := db.Exec(sqlstr, cc.Name)
	return err
}

// ChallengeCategories retrieves every ChallengeCategory from the database.
func ChallengeCategories(db DB) ([]ChallengeCategory, error) {
	const sqlstr = `SELECT name FROM cyboard.challenge_category`
	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ccs := []ChallengeCategory{}
	for rows.Next() {
		cc := ChallengeCategory{}
		if err = rows.Scan(&cc.Name); err != nil {
			return nil, err
		}
		ccs = append(ccs, cc)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ccs, nil
}
