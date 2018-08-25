package models

import (
	"testing"
	"time"

	"github.com/pereztr5/cyboard/server/apptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LatestServiceCheckRun(t *testing.T) {
	prepareTestDatabase(t)
	time_str := "2018-07-29T09:15:00.000-04:00"
	expected_ts, _ := time.Parse(time.RFC3339, time_str)

	ts, err := LatestServiceCheckRun(db)
	if assert.Nil(t, err) {
		assert.Equal(t, expected_ts, ts)
	}

	_, err = apptest.StdlibDB.Exec(`DELETE FROM service_check`)
	require.Nil(t, err)

	time_str = "1970-01-01T00:00:00.000Z"
	expected_ts, err = time.Parse(time.RFC3339, time_str)
	require.Nil(t, err)

	ts, err = LatestServiceCheckRun(db)
	if assert.Nil(t, err) {
		assert.True(t, expected_ts.Equal(ts), "Before service scoring starts, latest run should be considered unix epoch.")
	}
}

func Test_TeamServiceStatuses(t *testing.T) {
	prepareTestDatabase(t)

	expected := []TeamServiceStatusesView{
		{ServiceID: 1, ServiceName: "ping", Statuses: []ExitStatus{ExitStatusPass, ExitStatusPartial}},
	}
	// 'status' is pluralized like moose or something, right?
	statooses, err := TeamServiceStatuses(db)
	if assert.Nil(t, err) {
		require.Equal(t, expected, statooses, "Status of each monitored service for each team.")

		t.Run("batch insert of service statuses", func(t *testing.T) {
			time_str := "3000-01-01T00:00:00.000-04:00" // The future!
			ts, _ := time.Parse(time.RFC3339, time_str)

			checkResults := ServiceCheckSlice([]ServiceCheck{
				{CreatedAt: ts, TeamID: 1, ServiceID: 1, Status: ExitStatusFail, ExitCode: 127},
				{CreatedAt: ts, TeamID: 2, ServiceID: 1, Status: ExitStatusFail, ExitCode: 127},
			})

			err := checkResults.Insert(db)
			require.Nil(t, err)

			expect_failures := []TeamServiceStatusesView{
				{ServiceID: 1, ServiceName: "ping", Statuses: []ExitStatus{ExitStatusFail, ExitStatusFail}},
			}
			service_statuses, err := TeamServiceStatuses(db)
			if assert.Nil(t, err) {
				require.Equal(t, expect_failures, service_statuses, "All services should be failing.")
			}
		})
	}
}
