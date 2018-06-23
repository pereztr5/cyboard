package models

// TeamServiceStatusesResponse represents a service's status (pass, fail, timeout)
// for one team.
type TeamServiceStatusesResponse struct {
	TeamID      int        `json:"team_id"`
	Name        string     `json:"name"`
	ServiceID   int        `json:"service_id"`
	ServiceName string     `json:"service_name"`
	Status      ExitStatus `json:"status"`
}

// TeamServiceStatuses gets the service uptime status (pass, fail, timeout)
// for each team, for each service.
func TeamServiceStatuses(db DB) []TeamServiceStatusesResponse {
	const sqlstr = `
	SELECT id, name, ss.service, ss.service_name, ss.status
	FROM blueteam AS team,
	LATERAL (SELECT id AS service, name AS service_name, COALESCE(last(sc.status, sc.created_at), 'timeout') AS status
		FROM service
			LEFT JOIN service_check AS sc ON id = sc.service_id AND team.id = sc.team_id
		GROUP BY id, name
		ORDER BY id) AS ss`
}
