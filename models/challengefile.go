// Package models contains the types for schema 'cyboard'.
package models

// ChallengeFile represents a row from 'cyboard.challenge_file'.
type ChallengeFile struct {
	ID          int    `json:"id"`           // id
	ChallengeID int    `json:"challenge_id"` // challenge_id
	Filename    string `json:"filename"`     // filename
	Description string `json:"description"`  // description
}

// Insert inserts the ChallengeFile to the database.
func (cf *ChallengeFile) Insert(db DB) error {
	const sqlstr = `INSERT INTO cyboard.challenge_file (` +
		`challenge_id, filename, description` +
		`) VALUES (` +
		`$1, $2, $3` +
		`) RETURNING id`

	return db.QueryRow(sqlstr, cf.ChallengeID, cf.Filename, cf.Description).Scan(&cf.ID)
}

// Update updates the ChallengeFile in the database.
func (cf *ChallengeFile) Update(db DB) error {
	const sqlstr = `UPDATE cyboard.challenge_file SET (` +
		`challenge_id, filename, description` +
		`) = ( ` +
		`$2, $3, $4` +
		`) WHERE id = $1`

	_, err := db.Exec(sqlstr, cf.ID, cf.ChallengeID, cf.Filename, cf.Description)
	return err
}

// Delete deletes the ChallengeFile from the database.
func (cf *ChallengeFile) Delete(db DB) error {
	const sqlstr = `DELETE FROM cyboard.challenge_file WHERE id = $1`

	_, err := db.Exec(sqlstr, cf.ID)
	return err
}

// Challenge returns the Challenge associated with the ChallengeFile's ChallengeID (challenge_id).
func (cf *ChallengeFile) Challenge(db DB) (*Challenge, error) {
	return ChallengeByID(db, cf.ChallengeID)
}

// ChallengeFileByID retrieves a row from 'cyboard.challenge_file' as a ChallengeFile.
func ChallengeFileByID(db DB, id int) (*ChallengeFile, error) {
	const sqlstr = `SELECT ` +
		`id, challenge_id, filename, description ` +
		`FROM cyboard.challenge_file ` +
		`WHERE id = $1`

	cf := ChallengeFile{}

	err := db.QueryRow(sqlstr, id).Scan(&cf.ID, &cf.ChallengeID, &cf.Filename, &cf.Description)
	if err != nil {
		return nil, err
	}

	return &cf, nil
}
