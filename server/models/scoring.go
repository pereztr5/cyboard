package models

type TeamsScoresResponse struct {
	TeamID  int     `json:"team_id"`
	Name    string  `json:"name"`
	Score   float32 `json:"score"`
	Service float32 `json:"service"`
	Ctf     float32 `json:"ctf"`
	Other   float32 `json:"other"`
}

func TeamsScores(db DB) {
	const sqlstr = `
	SELECT
		team.id AS team_id,
		team.name AS team_name,
		service_score.points + ctf_score.points + other_score.points AS score,
		service_score.points AS service,
		ctf_score.points AS ctf,
		other_score.points AS other
	FROM team
		JOIN service_score ON team.id = service_score.team_id
		JOIN ctf_score ON team.id = ctf_score.team_id
		JOIN other_score ON team.id = other_score.team_id
	ORDER BY team.id`

}
