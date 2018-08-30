package server

import (
	"testing"
	"time"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/stretchr/testify/assert"
)

func timeMustParse(timestr string) time.Time {
	expected_ts, err := time.Parse(time.RFC3339, timestr)
	if err != nil {
		panic(err)
	}
	return expected_ts
}

func Test_CalcPointsPerCheck(t *testing.T) {
	cases := []struct {
		testname string

		event         *EventSettings
		checkInterval time.Duration
		srv           *models.Service

		expected float32
	}{
		{
			testname: "1hr_no_breaks",

			srv: &models.Service{
				StartsAt:    timeMustParse("2018-08-31T20:00:00-04:00"),
				TotalPoints: 100,
			},
			event: &EventSettings{
				End: timeMustParse("2018-08-31T21:00:00-04:00"),
			},
			checkInterval: time.Minute,

			// 100pts / (1hr / 1mins) = 100 pts / 60 checks = 1.66...
			expected: 1.666666666666666666666666666,
		},
		{
			testname: "8hrs_with_one_legit_break",

			srv: &models.Service{
				StartsAt:    timeMustParse("2018-09-01T00:00:00Z"),
				TotalPoints: 1000.0,
			},
			event: &EventSettings{
				Start: timeMustParse("2018-08-31T16:00:00Z"),
				End:   timeMustParse("2018-09-01T08:00:00Z"),
				Breaks: []ScheduledBreak{
					// First break occurs before the service starts, it is ignored.
					{
						StartsAt: timeMustParse("2018-08-31T20:00:00Z"),
						GoesFor:  2 * time.Hour,
					},
					{
						StartsAt: timeMustParse("2018-09-01T03:00:00Z"),
						GoesFor:  time.Hour + (15 * time.Minute),
					}},
			},
			checkInterval: 15 * time.Second,

			/* This is the example I did by hand weeks ago:
			   8h - 1.25h = 6.75 hrs; 24300 seconds
			   24300s / 15s = 1620 checks
			   1000.0 pts / 1620 chks = 0.617~ points per check */
			expected: 0.616666666666666666666666666,
		},
	}

	for _, tt := range cases {
		t.Run(tt.testname, func(t *testing.T) {
			got := CalcPointsPerCheck(tt.srv, tt.event, tt.checkInterval)
			assert.InDelta(t, tt.expected, got, 1e-3)
		})
	}
}
