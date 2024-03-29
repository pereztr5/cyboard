package models

import "time"

type TeamsScoresResponse struct {
	TeamID  int    `json:"team_id"`
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Service int    `json:"service"`
	Ctf     int    `json:"ctf"`
	Other   int    `json:"other"`
}

// TeamsScores TODO
func TeamsScores(db DB) ([]TeamsScoresResponse, error) {
	const sqlstr = `
	SELECT
		team.id AS team_id,
		team.name AS team_name,
		round(service_score.points + ctf_score.points + other_score.points) AS score,
		round(service_score.points) AS service,
		round(ctf_score.points) AS ctf,
		round(other_score.points) AS other
	FROM team
		JOIN service_score ON team.id = service_score.team_id
		JOIN ctf_score ON team.id = ctf_score.team_id
		JOIN other_score ON team.id = other_score.team_id
	ORDER BY team.id`

	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scores := []TeamsScoresResponse{}
	for rows.Next() {
		s := TeamsScoresResponse{}
		if err = rows.Scan(&s.TeamID, &s.Name, &s.Score, &s.Service, &s.Ctf, &s.Other); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return scores, nil
}

// LatestScoreChange retrieves the timestamp of the last event that changed any team's score.
// This is a lightweight way of checking if other pieces of info need updating.
// If the timestamp changes between calls, then that means other score or status related
// data should be queried for again.
func LatestScoreChange(db DB) (time.Time, error) {
	const sqlstr = `SELECT 'epoch' AS created_at
	UNION ALL SELECT created_at FROM service_check
	UNION ALL SELECT created_at FROM ctf_solve
	UNION ALL SELECT created_at FROM other_points
	ORDER BY created_at DESC
	LIMIT 1`
	var timestamp time.Time
	err := db.QueryRow(sqlstr).Scan(&timestamp)
	return timestamp, err
}
