package models

func TeamServiceScores(db DB) {
	const sqlstr = `
	SELECT service.name, team.name, sum(points)
	FROM service
		JOIN service_check AS sc ON id = sc.service_id
		JOIN team ON team.id = sc.team_id
	WHERE sc.status = 'pass'
	GROUP BY team.name, service.name
	ORDER BY team.name, service.name`
}

func TeamScore(db DB, teamID int) {
	const sqlstr = `
	SELECT service.name, sum(points)
	FROM service
		JOIN service_check AS sc ON id = sc.service_id
	JOIN team ON team.id = sc.team_id
	WHERE sc.status = 'pass' AND team.id = $1
	GROUP BY service.name ORDER BY service.name`
}

func TeamsScores(db DB) {
	const sqlstr = `
	WITH sv_score AS (
		SELECT team.id, sum(sv.points)
			FROM team
			JOIN service_check AS sc ON team.id = sc.team_id
			JOIN service AS sv ON sv.id = sc.service_id
		WHERE sc.status = 'pass'
		GROUP BY team.id
	),
	ctf_score AS (
		SELECT team.id, sum(ch.total)
		FROM team
			JOIN ctf_solve AS cs ON team.id = cs.team_id
			JOIN challenge AS ch ON ch.id = cs.challenge_id
		GROUP BY team.id
	),
	other_score AS (
		SELECT team.id, sum(o.points)
		FROM team
			JOIN other_score AS o ON team.id = o.team_id
		GROUP BY team.id
	)
	SELECT
		team.id AS team_id,
		COALESCE(sv_score.sum, 0) AS service_score,
		COALESCE(ctf_score.sum, 0) AS ctf_score,
		COALESCE(other_score.sum, 0) AS other_score
	FROM team
	LEFT JOIN sv_score ON team.id = sv_score.id
	LEFT JOIN ctf_score ON team.id = ctf_score.id
	LEFT JOIN other_score ON team.id = other_score.id
	ORDER BY team.id`

}
