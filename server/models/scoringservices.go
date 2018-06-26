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
// This explicitly requires a `pgx.ConnPool`, as it's batch insert method is
// completely different from any other SQL abstraction's methods.
func (sc ServiceCheckSlice) Insert(db *pgx.ConnPool) error {
	_, err := db.CopyFrom(
		serviceCheckTableIdent,
		serviceCheckTableColumns,
		&serviceCheckCopyFromRows{rows: sc},
	)
	return err
}

// LatestServiceCheckRun retrieves the timestamp of the last run of the service monitor.
// See: `LatestScoreChange` in `scoring.go`. This delta check is specific to services.
func LatestServiceCheckRun(db DB) (time.Time, error) {
	const sqlstr = `SELECT created_at FROM service_check UNION ALL '-infinity' ORDER BY created_at DESC LIMIT 1`
	var timestamp time.Time
	err := db.QueryRow(sqlstr).Scan(&timestamp)
	return timestamp, err
}

// TeamServiceStatusesView represents the current, calculated
// service's status (pass, fail, timeout), for one team.
type TeamServiceStatusesView struct {
	TeamID      int        `json:"team_id"`
	Name        string     `json:"name"`
	ServiceID   int        `json:"service_id"`
	ServiceName string     `json:"service_name"`
	Status      ExitStatus `json:"status"`
}

// TeamServiceStatuses gets the current service status (pass, fail, timeout)
// for each team, for each service.
func TeamServiceStatuses(db DB) ([]TeamServiceStatusesView, error) {
	const sqlstr = `
	SELECT id, name, ss.service, ss.service_name, ss.status
	FROM blueteam AS team,
	LATERAL (SELECT id AS service, name AS service_name, COALESCE(last(sc.status, sc.created_at), 'timeout') AS status
		FROM service
			LEFT JOIN service_check AS sc ON id = sc.service_id AND team.id = sc.team_id
		GROUP BY id, name
		ORDER BY id) AS ss`
	rows, err := db.Query(sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	xs := []TeamServiceStatusesView{}
	for rows.Next() {
		x := TeamServiceStatusesView{}
		if err = rows.Scan(&x.TeamID, &x.Name, &x.ServiceID, &x.ServiceName, &x.Status); err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return xs, nil
}
