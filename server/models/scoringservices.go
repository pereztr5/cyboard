package models

import (
	"time"

	"github.com/jackc/pgx"
)

// ServiceCheck represents a row from 'cyboard.service_check'.
type ServiceCheck struct {
	CreatedAt time.Time  `json:"created_at"` // created_at
	TeamID    int        `json:"team_id"`    // team_id
	ServiceID int        `json:"service_id"` // service_id
	Status    ExitStatus `json:"status"`     // status
	ExitCode  int16      `json:"exit_code"`  // exit_code
}

var (
	serviceCheckTableIdent   = pgx.Identifier{"cyboard", "service_check"}
	serviceCheckTableColumns = []string{
		"created_at", "team_id", "service_id", "status", "exit_code",
	}
)

// ServiceCheckSlice is an array of ServiceChecks, suitable to insert many of at once.
type ServiceCheckSlice []ServiceCheck

// serviceCheckCopyFromRows implements the pgx.CopyFromSource interface, allowing
// a ServiceCheckSlice to be inserted into the database.
type serviceCheckCopyFromRows struct {
	rows ServiceCheckSlice
	idx  int
}

func (ctr *serviceCheckCopyFromRows) Next() bool {
	ctr.idx++
	return ctr.idx < len(ctr.rows)
}

func (ctr *serviceCheckCopyFromRows) Values() ([]interface{}, error) {
	sc := ctr.rows[ctr.idx]
	val := []interface{}{
		sc.CreatedAt, sc.TeamID, sc.ServiceID, sc.Status, sc.ExitCode,
	}
	return val, nil
}

func (ctr *serviceCheckCopyFromRows) Err() error {
	return nil
}

// Insert a batch of service monitor results efficiently into the database.
func (sc ServiceCheckSlice) Insert(db DB) error {
	_, err := db.CopyFrom(
		serviceCheckTableIdent,
		serviceCheckTableColumns,
		&serviceCheckCopyFromRows{rows: sc, idx: -1},
	)
	return err
}

// LatestServiceCheckRun retrieves the timestamp of the last run of the service monitor.
// See: `LatestScoreChange` in `scoring.go`. This delta check is specific to services.
func LatestServiceCheckRun(db DB) (time.Time, error) {
	const sqlstr = `SELECT created_at FROM service_check UNION ALL SELECT 'epoch' ORDER BY created_at DESC LIMIT 1`
	var timestamp time.Time
	err := db.QueryRow(sqlstr).Scan(&timestamp)
	return timestamp, err
}

// TeamServiceStatusesView represents the current, calculated
// service's status (pass, fail, timeout), for all teams.
type TeamServiceStatusesView struct {
	ServiceID   int          `json:"service_id"`
	ServiceName string       `json:"service_name"`
	Statuses    []ExitStatus `json:"statuses"`
}

// TeamServiceStatuses gets the current service status (pass, fail, timeout)
// for each team, for each non-disabled service.
func TeamServiceStatuses(db DB) ([]TeamServiceStatusesView, error) {
	const sqlstr = `
	SELECT ss.service, ss.service_name, jsonb_agg(ss.status) AS statuses
	FROM blueteam AS team,
	LATERAL (SELECT id AS service, name AS service_name, COALESCE(last(sc.status, sc.created_at), 'timeout') AS status
		FROM service
			LEFT JOIN service_check AS sc ON id = sc.service_id AND team.id = sc.team_id
		WHERE service.disabled = false
		GROUP BY service.id, team.name
		ORDER BY service.id, team.id) AS ss
	GROUP BY ss.service, ss.service_name`
	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []TeamServiceStatusesView{}
	for rows.Next() {
		x := TeamServiceStatusesView{}
		if err = rows.Scan(&x.ServiceID, &x.ServiceName, &x.Statuses); err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}
