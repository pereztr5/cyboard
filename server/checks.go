package server

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pereztr5/cyboard/server/models"
)

type Check struct {
	*models.Service
	Command *exec.Cmd
}

func (c *Check) String() string {
	return fmt.Sprintf(`Check{name=%q, fullcmd="%s %s", pts=%v}`,
		c.Name, filepath.Base(c.Command.Path), c.Args, c.Points)
}

func getScript(path string) (*exec.Cmd, error) {
	dir, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	_, err = exec.LookPath(path)
	if err != nil {
		return nil, err
	}
	return exec.Command(dir), nil
}

func prepareChecks(services []models.Service, scriptsDir string) []Check {
	checks := []Check{}

	for idx, service := range services {
		if service.Disabled {
			Logger.Warnf("check.%d: DISABLED.", idx)
			continue
		}

		script, err := getScript(filepath.Join(scriptsDir, service.Script))
		if err != nil {
			Logger.Warnf("check.%d: SKIPPING! Failed to locate script: %v", idx, err)
			continue
		}
		checks = append(checks, Check{Service: &service, Command: script})
	}

	Logger.Print("All services:")
	for i, check := range checks {
		Logger.Printf("  [%d] %v", i, &check)
	}

	return checks
}

func parseArgs(name string, args string, ip int) []string {
	//TODO: Quick fix but need to com back and do this right
	//TODO(pereztr): This should at least have to be surrounded in braces or some meta-chars
	const ReplacementText = "IP"
	ipStr := "192.168.0." + strconv.Itoa(ip)
	nArgs := name + " " + strings.Replace(args, ReplacementText, ipStr, 1)
	return strings.Split(nArgs, " ")
}

func runCmd(team *models.BlueteamView, check *Check, timestamp time.Time, timeout time.Duration, status chan models.ServiceCheck) {
	// TODO: Currently only one IP per team is supported
	cmd := *check.Command
	cmd.Args = parseArgs(cmd.Path, strings.Join(check.Args, " "), int(team.BlueteamIP))

	err := cmd.Start()

	result := models.ServiceCheck{
		CreatedAt: timestamp,
		TeamID:    team.ID,
		ServiceID: check.ID,
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
				"service": check.Name,
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
	Teams     []models.BlueteamView

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

func (m *Monitor) ReloadServicesAndTeams(checksDir string) {
	m.Lock()
	defer m.Unlock()

	services, teams, err := models.LoadServicesAndTeams(db)
	if err != nil {
		// Postgres is busted, someone is going to have to look at this, resolve it by hand
		Logger.WithError(err).Fatal("failed to get teams & services for service monitor")
		return
	}
	m.Teams = teams

	checks := prepareChecks(services, checksDir)
	// Realloc check slices. Anticipate most checks will be started, so alloc accordingly.
	m.Checks = make([]Check, 0, len(checks))
	m.Unstarted = make([]Check, 0)

	now := time.Now()
	for _, c := range checks {
		if now.After(c.StartsAt) {
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

		// m.Checks & m.Teams need protection from concurrent use.
		// The PG Listen thread updates them whenever the DB changes.
		m.Lock()

		// Each check has a separte time for when they should begin, so examine each of them
		// and if they are now past their start time, append them to the active checks.
		for i := 0; i < len(m.Unstarted); i++ {
			c := m.Unstarted[i]
			if now.After(c.StartsAt) {
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

		if len(resultsBuf) != len(m.Teams)*len(m.Checks) {
			resultsBuf = make([]models.ServiceCheck, len(m.Teams)*len(m.Checks))
		}

		log.Infof("Running [%d] Checks. Started +jitter = %s +%v",
			len(m.Checks), now.Format(time.RFC3339), jitter.Truncate(time.Millisecond))

		// Run each check against each teams' infrastructure.
		// TODO: Add WorkGroup/timeout safety, in case runCmd never acks back
		for _, team := range m.Teams {
			for _, check := range m.Checks {
				go runCmd(&team, &check, now, srvmon.Timeout, status)
			}
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

func (m *Monitor) ListenForConfigUpdatesFromPG(ctx context.Context, checksDir string) {
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

		m.ReloadServicesAndTeams(checksDir)
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

	monitor.ReloadServicesAndTeams(checksDir)

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

	go monitor.ListenForConfigUpdatesFromPG(ctx, checksDir)
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
