package monitor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/pereztr5/cyboard/server"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

const PGListenNotifyChannel = "cyboard.server.checks"

type CheckResults struct {
	Results []models.ServiceCheck
	Err     error
}

// monitorRetryBackoffFn defines how long to wait in between attempts at critical
// database operations. Right now, this will sleep the thread for attempt^2 seconds
func monitorRetryWithBackoff(fn func() error) error {
	const monitorDBMaxRetries = 5

	var err error
	for attempt := 1; attempt <= monitorDBMaxRetries; attempt++ {
		x := time.Duration(attempt) // satisfy the compiler
		time.Sleep(x * x * time.Second)
		err = fn()
		if err == nil {
			Logger.WithField("attempts", attempt).Warn("failed to contact db several times, but managed to pull through!")
			break
		}
	}
	return err
}

type Monitor struct {
	Checks    []models.MonitorService
	Unstarted []models.MonitorService
	Teams     []models.BlueteamView

	breaktimeC chan time.Duration
	done       chan struct{}

	rando *rand.Rand
	*sync.Mutex
}

func NewMonitor() *Monitor {
	return &Monitor{
		Mutex:      new(sync.Mutex),
		breaktimeC: make(chan time.Duration),
		done:       make(chan struct{}),
		rando:      rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
	}
}

func (m *Monitor) Stop() {
	close(m.done)
}

func (m *Monitor) ReloadTeamsAndServices(baseIP string) {
	m.Lock()
	defer m.Unlock()

	services, err := models.MonitorServices(db)
	if err != nil {
		err = monitorRetryWithBackoff(func() error {
			var err error
			services, err = models.MonitorServices(db)
			return err
		})
		if err != nil {
			// Postgres is busted, someone is going to have to look at this, resolve it by hand
			Logger.WithError(err).Fatal("failed to get services for service monitor")
			return
		}
	}

	m.Teams, err = models.AllBlueteams(db)
	if err != nil {
		Logger.WithError(err).Fatal("failed to get teams for service monitor")
		return
	}

	checks := prepareServices(services, baseIP)
	// Realloc check slices. Anticipate most checks will be started, so alloc accordingly.
	m.Checks = make([]models.MonitorService, 0, len(checks))

	// Queue every check as unstarted, then the scheduler will move the started ones over to
	// m.Checks and then it knows to signal the task workers to update their checks/teams.
	m.Unstarted = checks
}

func (m *Monitor) Run(event *EventSettings, srvmon *ServiceMonitorSettings) {
	log := Logger.WithField("thread", "monitor_checks")

	// Run command every x seconds until scheduled end time.
	checkTicker := time.NewTicker(srvmon.Intervals)

	// delta between runs & timeout, for jittering checks.
	freeTime := server.Int64Max(int64(srvmon.Intervals-srvmon.Timeout), 1)

	// Update redis timeout; This is how workers know their deadlines.
	redisUpdateTimeout(rstore, srvmon.Timeout)

	// Max time to wait for results to come back from workers via redis
	// note: Division here truncates down, so a second is added to provide leeway
	const leeway = 100 * time.Millisecond
	recvTimeout := ((srvmon.Timeout + leeway).Nanoseconds() / int64(time.Second)) + 1

	// Scripts get executed in a separate goroutine, one for each team, for each service,
	// and then send results back on the status channel to the resultsBuf.
	statusChan := make(chan CheckResults)
	var resultsBuf []models.ServiceCheck

	// waitToContinue handles the check runner scheduler, break scheduler, and done signals.
	// It should always wait either the Interval between check runs (checkTicker), or
	// for the duration of the next scheduled break (sent by the BreaktimeScheduler),
	// unless the monitor is stopping.
	//
	// Returns true when the Check Runner should continue, or false to signal a stop.
	waitToContinue := func() bool {
		select {
		case <-checkTicker.C:
		case breakDuration := <-m.breaktimeC:
			checkTicker.Stop()
			log.WithField("duration", breakDuration).
				Infof("Break has begun, monitor is paused during break.")
			select {
			case <-time.After(breakDuration):
				checkTicker = time.NewTicker(srvmon.Intervals)
			case <-m.done:
				return false
			}
		case <-m.done:
			return false
		}
		return true
	}

	// Start the daemon!
	for {
		now := time.Now()
		cmdSig := coordination.SignalRun

		// Add unpredictability to the service checking by waiting some time afterwards.
		jitter := time.Duration(m.rando.Int63n(freeTime))
		<-time.After(jitter)

		// m.Checks needs protection from concurrent use.
		// The PG Listen thread updates them whenever the DB changes.
		m.Lock()

		// Each check has a separate time for when they should begin, so examine each of them
		// and if they are now past their start time, append them to the active checks.
		for i := 0; i < len(m.Unstarted); i++ {
			c := m.Unstarted[i]
			if now.After(c.StartsAt) {
				m.Checks = append(m.Checks, c)
				m.Unstarted = append(m.Unstarted[:i], m.Unstarted[i+1:]...)
				i--

				redisUpdateTeamsAndServices(rstore, m.Teams, m.Checks)
				cmdSig = coordination.SignalReloadThenRun
			}
		}

		// If there's no checks to run, just wait until they get set up in the pg-listener.
		if len(m.Checks) == 0 {
			m.Unlock()
			if !waitToContinue() {
				break
			}
			continue
		}

		nResults := len(m.Checks) * len(m.Teams)
		if len(resultsBuf) != nResults {
			resultsBuf = make([]models.ServiceCheck, nResults)
		}

		log.Infof("Scheduling [%d] Checks. Started +jitter = %s +%v",
			nResults, now.Format(time.RFC3339), jitter.Truncate(time.Millisecond))

		// Run each check against each teams' infrastructure.

		// Flush any uncollected results, then signal workers to run checks
		{
			c := rstore.Get()
			for _, team := range m.Teams {
				c.Send("DEL", coordination.FmtResultsKey(team.BlueteamIP))
			}
			c.Send("PUBLISH", coordination.RedisKeySchedule, cmdSig)
			c.Flush()
			c.Close()
		}

		for _, team := range m.Teams {
			go receiveResults(statusChan, rstore, recvTimeout, &now, coordination.FmtResultsKey(team.BlueteamIP))
		}

		// One second is taken away to provide leeway to synchronize between host & workers.
		deadline := time.After(srvmon.Timeout + time.Duration(freeTime) - jitter)

		var workerErrs ErrorSlice
		for i := range m.Teams {
			select {
			case <-deadline:
				// goroutines not responding; should never happen
				workerErrs = append(workerErrs,
					fmt.Errorf("Thread/goroutine failed to ack back in time!"))
			case results := <-statusChan:
				if results.Err != nil {
					workerErrs = append(workerErrs, results.Err)
				} else {
					j := i * len(m.Checks)
					max := j + len(m.Checks)
					if len(resultsBuf) < max {
						workerErrs = append(workerErrs,
							fmt.Errorf("Fatal programmer error: detected buffer overflow: "+
								"len(resultsBuf)=%d, max=%d", nResults, max))
					} else {
						subslice := resultsBuf[j:max]
						copy(subslice, results.Results)
					}
				}
			}
		}

		for i := range resultsBuf {
			resultsBuf[i].CreatedAt = now
		}

		m.Unlock()

		if len(workerErrs) > 0 {
			Logger.WithError(workerErrs).Error("!! ALL CHECKS SKIPPED !! Due to fatal worker error")
		} else if err := models.ServiceCheckSlice(resultsBuf).DummyInsert(db); err != nil {
			// Try *really hard* to not lose unrecoverable scoring data.
			err = monitorRetryWithBackoff(func() error {
				err := models.ServiceCheckSlice(resultsBuf).Insert(db)
				return err
			})
			if err != nil {
				log.WithError(err).Error("failed to insert service results despite multiple attempts!")
			}
		}

		if !waitToContinue() {
			break
		}
	}

	// cleanup
	checkTicker.Stop()
}

func (m *Monitor) ListenForConfigUpdatesFromPG(ctx context.Context, baseIP string) {
	log := Logger.WithField("thread", "monitor_pg-listener")

	conn, err := rawDB.Acquire()
	if err != nil {
		log.WithError(err).Fatal("failed to get pg connection")
		return
	}
	defer rawDB.Release(conn)
	defer conn.Unlisten(PGListenNotifyChannel)

	err = conn.Listen(PGListenNotifyChannel)
	if err != nil {
		log.WithError(err).Fatal("failed to call sql LISTEN")
		return
	}

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			// likely context cancellation
			log.WithError(err).WithField("notification", notification).
				Debugf("error while listening for notification")
			return
		}

		log.WithField("notif", notification).Debug("update received")

		m.ReloadTeamsAndServices(baseIP)
		log.Info("Settings reloaded!")
	}
}

func (m *Monitor) BreaktimeScheduler(breaks []ScheduledBreak) {
	log := Logger.WithField("thread", "breaktime_scheduler")
	for {
		// `break` is also a syntactical keyword, so this is gonna get messy
		var nextbreak *ScheduledBreak
		now := time.Now()
		for _, br := range breaks {
			if now.Before(br.StartsAt) {
				nextbreak = &br
			} else if now.After(br.StartsAt) && now.Before(br.End()) {
				nextbreak = &br
			}
		}
		if nextbreak == nil {
			// No more breaks left, we're done here
			return
		}

		wait := time.Until(nextbreak.StartsAt)
		log.WithField("at", nextbreak.StartsAt).WithField("in", wait).Info("Next break")

		<-time.After(wait)

		endtime := nextbreak.End()
		breakDuration := time.Until(endtime)
		// Let the main Run thread know how long to pause for.
		log.WithField("ends at", endtime).Info("Break started!")
		m.breaktimeC <- breakDuration
		// The scheduler itself should pause until the break is over.
		<-time.After(time.Until(endtime))
	}
}
