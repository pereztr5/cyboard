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
	"strconv"
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
		return fmt.Sprintf(`Check{team=%+v, service=%+v, <unfinished>}`, c.Team, c.Service)
	} else {
		return fmt.Sprintf(`Check{team=%+v, service=%+v, command=%+v}`, c.Team, c.Service, c.Command)
	}
}

func getScript(path string) (*exec.Cmd, error) {
	execpath, err := exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	abspath, err := filepath.Abs(execpath)
	if err != nil {
		return nil, err
	}
	return exec.Command(abspath), nil
}

func prepareChecks(teamsAndServices []models.MonitorTeamService, scriptsDir, baseIP string) []Check {
	checks := []Check{}

	for i := range teamsAndServices {
		tas := &teamsAndServices[i]

		path := filepath.Join(scriptsDir, tas.Service.Script)
		script, err := getScript(path)
		if err != nil {
			Logger.Warnf("check.%d (name=%q): SKIPPING! Failed to locate script: %v",
				tas.Service.ID, tas.Service.Name, err)
			continue
		}
		script.Dir = scriptsDir

		// Set args with team's IP, Name, and ID using simple string replace on each argument
		var s string
		script.Args = make([]string, 0, len(tas.Service.Args))
		for _, arg := range tas.Service.Args {
			switch arg {
			case "{IP}":
				// BaseIP is a config.toml option that looks like "192.168.0." which we add
				// the last octet to from the `cyboard.team` table, giving us the
				// full ip. E.G. "192.168.0.7"
				s = baseIP + strconv.FormatInt(int64(tas.Team.IP), 10)
			case "{TEAM_4TH_OCTET}":
				s = strconv.FormatInt(int64(tas.Team.IP), 10)
			case "{TEAM_NAME}":
				s = tas.Team.Name
			case "{TEAM_ID}":
				s = strconv.FormatInt(int64(tas.Team.ID), 10)
			default:
				s = arg
			}
			script.Args = append(script.Args, s)
		}

		checks = append(checks, Check{MonitorTeamService: tas, Command: script})
	}

	// Print all services from the Checks.
	//
	// (hoop-jumping required because the checks are a cross product of all teams x services,
	// so we need to filter it down to just services and only print the service bits.)
	if Logger.IsLevelEnabled(logrus.InfoLevel) {
		// filter to unique services, by service id.
		uniqServices := map[int]Check{}
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
			c := uniqServices[id]
			Logger.Infof(`  [%d] Check{name=%q, fullcmd="%s %s"}`,
				c.Service.ID, c.Service.Name,
				filepath.Base(c.Command.Path), strings.Join(c.Service.Args, " "))
		}
	}

	return checks
}

func getCmdResult(cmd *exec.Cmd, timeout time.Duration) (int16, models.ExitStatus) {
	var code int16
	var status models.ExitStatus

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			Logger.WithFields(logrus.Fields{
				"error":  err.Error(),
				"script": cmd.Path,
			}).Error("Failed to Kill:")
		}
		code, status = 129, models.ExitStatusTimeout
	case <-done:
		// As long as it is done the error doesn't matter
		code = int16(cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus())

		switch code {
		case 0:
			status = models.ExitStatusPass
		case 1:
			status = models.ExitStatusPartial
		default:
			status = models.ExitStatusFail
		}
	}
	return code, status
}

func runCmd(check *Check, timestamp time.Time, timeout time.Duration, status chan models.ServiceCheck) {
	cmd := *check.Command

	result := models.ServiceCheck{
		CreatedAt: timestamp,
		TeamID:    check.Team.ID,
		ServiceID: check.Service.ID,
	}

	if err := cmd.Start(); err != nil {
		Logger.Error("Could not run script:", err)
		result.ExitCode = 127 // 127=command not found: http://www.tldp.org/LDP/abs/html/exitcodes.html
		result.Status = models.ExitStatusTimeout
		status <- result
		return
	}

	result.ExitCode, result.Status = getCmdResult(&cmd, timeout)
	status <- result
}

const PGListenNotifyChannel = "cyboard.server.checks"

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
	Checks    []Check
	Unstarted []Check

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

func (m *Monitor) ReloadServicesAndTeams(checksDir, baseIP string) {
	m.Lock()
	defer m.Unlock()

	teamsAndServices, err := models.MonitorTeamsAndServices(db)
	if err != nil {
		err = monitorRetryWithBackoff(func() error {
			var err error
			teamsAndServices, err = models.MonitorTeamsAndServices(db)
			return err
		})
		if err != nil {
			// Postgres is busted, someone is going to have to look at this, resolve it by hand
			Logger.WithError(err).Fatal("failed to get teams & services for service monitor")
			return
		}
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
		for i := range m.Checks {
			go runCmd(&m.Checks[i], now, srvmon.Timeout, status)
		}
		for idx := range resultsBuf {
			resultsBuf[idx] = <-status
		}

		m.Unlock()

		if err := models.ServiceCheckSlice(resultsBuf).Insert(db); err != nil {
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

func (m *Monitor) ListenForConfigUpdatesFromPG(ctx context.Context, checksDir, baseIP string) {
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

		select {
		case <-m.done:
			return
		case <-time.After(wait):
			endtime := nextbreak.End()
			breakDuration := time.Until(endtime)
			// Let the main Run thread know how long to pause for.
			log.WithField("ends at", endtime).Info("Break started!")
			select {
			case m.breaktimeC <- breakDuration:
				// The scheduler itself should pause until the break is over.
				select {
				case <-time.After(time.Until(endtime)):
					continue
				case <-m.done:
					return
				}
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
	- on break -> stop monitoring and pause everything other than the updater
	- startup during a scheduled break -> immediately pause
	- startup before the event begins -> immeditately pause
	- startup after the event is over -> immediately stop
	- end of event -> cancel everything and clean up
	- update to db -> reload teams & services
	- database errors -> retry a few times, then just straight die
	- magically dying goroutines -> cosmic anomaly, lose hope
	*/
	monitor := NewMonitor()
	defer monitor.Stop()

	monitor.ReloadServicesAndTeams(checksDir, baseIP)

	// Create control ctx, to let the pg-listener goroutine be stopped on demand.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Separate thread listens for updates from the DB and automatically reloads.
	go monitor.ListenForConfigUpdatesFromPG(ctx, checksDir, baseIP)

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

	now := time.Now()
	for _, br := range event.Breaks {
		endtime := br.End()
		if now.After(br.StartsAt) && now.Before(endtime) {
			breakDuration := time.Until(endtime)
			// Monitor was started during a break, pause immediately
			Logger.WithFields(logrus.Fields{
				"ends at":   endtime.Format(time.Stamp),
				"remaining": breakDuration,
			}).Infof("Waiting until break is over...")
			select {
			case <-time.After(breakDuration):
			case <-monitor.done:
				return
			case <-sigtermC:
				return
			}
		}
	}

	Logger.Println("Starting Checks")
	go monitor.BreaktimeScheduler(event.Breaks)
	go monitor.Run(&event, &checkCfg.ServiceMonitor)

	// Stop if: 1. The event is over 2. monitor has stopped 3. received ctrl+C (SIGTERM)
	select {
	case <-time.After(time.Until(event.End)):
		Logger.Info("Event is over. Done Checking Services")
	case <-monitor.done:
	case <-sigtermC:
	}
}
