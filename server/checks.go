package server

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pereztr5/cyboard/server/models"
)

type Check struct {
	*models.MonitorTeamService
	Command *exec.Cmd
}

func (c Check) String() string {
	if c.MonitorTeamService == nil {
		return "Check{}"
	} else if c.Command == nil {
		return fmt.Sprintf(`Check{name=%q, <unfinished>}`, c.Service.Name)
	} else {
		return fmt.Sprintf(`Check{name=%q, fullcmd="%s %s"}`,
			c.Service.Name, filepath.Base(c.Command.Path), strings.Join(c.Service.Args, " "))
	}
}

func getScript(path string, args []string) (*exec.Cmd, error) {
	dir, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	_, err = exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	return exec.Command(dir, args...), nil
}

func prepareChecks(teamsAndServices []models.MonitorTeamService, scriptsDir, baseIP string) []Check {
	checks := []Check{}

	for _, tas := range teamsAndServices {
		path := filepath.Join(scriptsDir, tas.Service.Script)
		script, err := getScript(path, tas.Service.Args)
		if err != nil {
			Logger.Warnf("check.%d (name=%q): SKIPPING! Failed to locate script: %v",
				tas.Service.ID, tas.Service.Name, err)
			continue
		}

		// Set args with team's IP, Name, and ID using simple string replace on each argument
		var s string
		script.Args = make([]string, 0, len(tas.Service.Args))
		for _, arg := range tas.Service.Args {
			switch arg {
			case "{IP}":
				// BaseIP is a config.toml option that looks like "192.168.0." which we add
				// the last octet to from the `cyboard.team` table, giving us the
				// full ip. E.G. "192.168.0.7"
				s = baseIP + string(tas.Team.IP)
			case "{TEAM_NAME}":
				s = tas.Team.Name
			case "{TEAM_ID}":
				s = string(tas.Team.ID)
			default:
				s = arg
			}
			script.Args = append(script.Args, s)
		}

		checks = append(checks, Check{MonitorTeamService: &tas, Command: script})
	}

	// Print all services from the Checks.
	//
	// (hoop-jumping required because the checks are a cross product of all teams x services,
	// so we need to filter it down to just services and only print the service bits.)
	if Logger.GetLevel() == logrus.InfoLevel {
		// filter to unique services, by service id.
		var uniqServices map[int]Check
		for _, check := range checks {
			uniqServices[check.Service.ID] = check
		}

		// get ordered IDs, to keep log statements ordered
		var ids []int
		for id := range uniqServices {
			ids = append(ids, id)
		}
		sort.Ints(ids)

		Logger.Info("All services:")
		for _, id := range ids {
			check := uniqServices[id]
			Logger.Infof("  [%d] %v", check.Service.ID, &check)
		}
	}

	return checks
}

func runCmd(check *Check, timestamp time.Time, timeout time.Duration, status chan models.ServiceCheck) {
	// TODO: Currently only one IP per team is supported
	cmd := *check.Command

	err := cmd.Start()

	result := models.ServiceCheck{
		CreatedAt: timestamp,
		TeamID:    check.Team.ID,
		ServiceID: check.Service.ID,
	}

	if err != nil {
		Logger.Error("Could not run script:", err)
		result.ExitCode = 127 // 127=command not found: http://www.tldp.org/LDP/abs/html/exitcodes.html
		result.Status = models.ExitStatusTimeout
		status <- result
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			Logger.WithFields(logrus.Fields{
				"error":   err,
				"service": check.Service.Name,
			}).Error("Failed to Kill:")
		}
		result.ExitCode = 129
		result.Status = models.ExitStatusTimeout
	case <-done:
		// As long as it is done the error doesn't matter
		result.ExitCode = int16(cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus())

		switch result.ExitCode {
		case 0:
			result.Status = models.ExitStatusPass
		case 1:
			result.Status = models.ExitStatusPartial
		default:
			result.Status = models.ExitStatusFail
		}
	}
	status <- result
}

const PGListenNotifyChannel = "cyboard.server.checks"

type Monitor struct {
	Checks    []Check
	Unstarted []Check

	breaktimeC chan time.Duration
	done       chan struct{}

	rando *rand.Rand
	*sync.Mutex
}

func NewMonitor() *Monitor {
	return &Monitor{
		breaktimeC: make(chan time.Duration),
		done:       make(chan struct{}),
		rando:      rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
	}
}

func (m *Monitor) Stop() {
	close(m.done)
}

func (m *Monitor) ReloadServicesAndTeams(checksDir, baseIP string) {
	m.Lock()
	defer m.Unlock()

	teamsAndServices, err := models.MonitorTeamsAndServices(db)
	if err != nil {
		// Postgres is busted, someone is going to have to look at this, resolve it by hand
		// TODO: Retry at least a few times before crashing
		Logger.WithError(err).Fatal("failed to get teams & services for service monitor")
		return
	}

	checks := prepareChecks(teamsAndServices, checksDir, baseIP)
	// Realloc check slices. Anticipate most checks will be started, so alloc accordingly.
	m.Checks = make([]Check, 0, len(checks))
	m.Unstarted = make([]Check, 0)

	now := time.Now()
	for _, c := range checks {
		if now.After(c.Service.StartsAt) {
			m.Checks = append(m.Checks, c)
		} else {
			m.Unstarted = append(m.Unstarted, c)
		}
	}

	if len(m.Checks) == 0 {
		Logger.Warn("No checks enabled/configured, waiting for services to be updated")
	}
}

func (m *Monitor) Run(event *EventSettings, srvmon *ServiceMonitorSettings) {
	log := Logger.WithField("thread", "monitor_checks")

	log.Println("Starting Checks")

	// Run command every x seconds until scheduled end time
	checkTicker := time.NewTicker(srvmon.Intervals)

	// delta between runs & timeout, for jittering checks.
	// There must be at least 1sec of jitter.
	freeTime := Int64Max(int64(srvmon.Intervals-srvmon.Timeout), 1)

	// Scripts get executed in a separate goroutine, one for each team, for each service,
	// and then send results back on the status channel to the resultsBuf.
	status := make(chan models.ServiceCheck)
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

		// Add unpredictability to the service checking by waiting some time afterwards.
		jitter := time.Duration(m.rando.Int63n(freeTime))
		<-time.After(jitter)

		// m.Checks needs protection from concurrent use.
		// The PG Listen thread updates them whenever the DB changes.
		m.Lock()

		// Each check has a separte time for when they should begin, so examine each of them
		// and if they are now past their start time, append them to the active checks.
		for i := 0; i < len(m.Unstarted); i++ {
			c := m.Unstarted[i]
			if now.After(c.Service.StartsAt) {
				m.Checks = append(m.Checks, c)
				m.Unstarted = append(m.Unstarted[:i], m.Unstarted[i+1:]...)
				i--
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

		if len(resultsBuf) != len(m.Checks) {
			resultsBuf = make([]models.ServiceCheck, len(m.Checks))
		}

		log.Infof("Running [%d] Checks. Started +jitter = %s +%v",
			len(m.Checks), now.Format(time.RFC3339), jitter.Truncate(time.Millisecond))

		// Run each check against each teams' infrastructure.
		// TODO: Add WorkGroup/timeout safety, in case runCmd never acks back
		for _, check := range m.Checks {
			go runCmd(&check, now, srvmon.Timeout, status)
		}
		for idx := range resultsBuf {
			resultsBuf[idx] = <-status
		}

		m.Unlock()

		if err := models.ServiceCheckSlice(resultsBuf).Insert(db); err != nil {
			log.WithError(err).Error("failed to insert service results")
		}

		if !waitToContinue() {
			break
		}
	}

	// cleanup
	checkTicker.Stop()
}

func (m *Monitor) ListenForConfigUpdatesFromPG(ctx context.Context, checksDir, baseIP string) {
	log := Logger.WithField("thread", "monitor_pg-listener")

	conn, err := rawDB.Acquire()
	if err != nil {
		log.WithError(err).Fatal("failed to get pg connection")
		return
	}
	defer conn.Close()
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

		m.ReloadServicesAndTeams(checksDir, baseIP)
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
			}
		}
		if nextbreak == nil {
			// No more breaks left, we're done here
			return
		}

		wait := time.Until(nextbreak.StartsAt)
		log.WithField("at", nextbreak.StartsAt).WithField("in", wait).Debug("next break")

		select {
		case <-m.done:
			return
		case <-time.After(wait):
			// Let the main Run thread know how long to pause for
			log.WithField("duration", nextbreak.GoesFor).Debug("break started!")
			select {
			case m.breaktimeC <- nextbreak.GoesFor:
				// Once signalled, go back to waiting until the next break
				continue
			case <-m.done:
				return
			}
		}
	}
}

func ChecksRun(checkCfg *Configuration) {
	SetupCheckServiceLogger(&checkCfg.Log)
	SetupPostgres(checkCfg.Database.URI)

	sigtermC := make(chan os.Signal, 1)
	signal.Notify(sigtermC, os.Interrupt)

	checksDir := checkCfg.ServiceMonitor.ChecksDir
	baseIP := checkCfg.ServiceMonitor.BaseIP
	event := checkCfg.Event

	/* lifecycle cases to handle:
	- regular -> no restart
	- startup -> init everything
	- on break, and setting up for the next break -> cancel all restart signals, ...then wait and restart
	- end of event -> cancel everything and clean up
	- update to db -> reload teams & services
	- database errors -> just straight die
	- magically dying goroutines -> cosmic anomaly, lose hope
	*/
	monitor := NewMonitor()
	defer monitor.Stop()

	monitor.ReloadServicesAndTeams(checksDir, baseIP)

	if time.Now().Before(event.Start) {
		Logger.Infof("Waiting until the event starts [%v]...",
			event.Start.Format(time.UnixDate))

		select {
		case <-time.After(time.Until(event.Start)):
		case <-monitor.done:
			return
		case <-sigtermC:
			return
		}
	}

	// Create control ctx, to let the pg-listener goroutine be stopped on demand.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.ListenForConfigUpdatesFromPG(ctx, checksDir, baseIP)
	go monitor.Run(&event, &checkCfg.ServiceMonitor)

	go monitor.BreaktimeScheduler(event.Breaks)

	// Stop if: 1. The event is over 2. monitor has stopped 3. received ctrl+C (SIGTERM)
	select {
	case <-time.After(time.Until(event.End)):
		Logger.Info("Event is over. Done Checking Services")
	case <-monitor.done:
	case <-sigtermC:
	}
}
