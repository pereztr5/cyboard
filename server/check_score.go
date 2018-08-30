package server

import (
	"time"

	"github.com/pereztr5/cyboard/server/models"
)

// CalcPointsPerCheck determines the fraction of points to award when a service
// check passes. Inputs `event` and `checkInterval` will come from `config.toml`.
// Returns a value suitable for the Service.Points field (presumably to be
// assigned to the parameter Service `srv`).
//
// The algorithm works as follows:
//
//   Span (difference) of time from service.StartsAt to event.End
//   Minus the time for each break that occurs within that timeframe
//   Then the number of checks that occur is the divison of that span by checkInterval
//   Finally, return the service.TotalPoints divided by the num of checks, which
//     is the points to award per check.
//
// Caveats:
// This doesn't handle the case a service is set to start during a break (shouldn't happen),
// and also won't account for overlapping breaks (also shouldn't happen.)
// Both of those cases are misconfigurations, that validation will prevent.
func CalcPointsPerCheck(
	srv *models.Service,
	event *EventSettings,
	checkInterval time.Duration,
) float32 {
	span := event.End.Sub(srv.StartsAt)

	for _, brk := range event.Breaks {
		if brk.StartsAt.After(srv.StartsAt) {
			span -= brk.GoesFor
		}
	}

	numChecks := int64(span / checkInterval)
	return srv.TotalPoints / float32(numChecks)
}
