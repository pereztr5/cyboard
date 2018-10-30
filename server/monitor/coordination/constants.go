package coordination

import "fmt"

const (
	SignalRun           = 1
	SignalReloadThenRun = 2

	RedisKeyTimeout  = "cy:check:timeout"
	RedisKeyTeams    = "cy:check:teams"
	RedisKeyServices = "cy:check:services"
	RedisKeySchedule = "cy:check:schedule"
	RedisKeyResults  = "cy:check:results"
)

func FmtResultsKey(ip int16) string {
	return fmt.Sprintf("%s:%d", RedisKeyResults, ip)
}
