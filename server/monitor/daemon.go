package monitor

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pereztr5/cyboard/server"
)

func ChecksRun(checkCfg *Configuration) {
	server.SetupCheckServiceLogger(&checkCfg.Log)
	Logger = server.Logger

	srvmon := checkCfg.ServiceMonitor
	setupRedis(srvmon.RedisConnType, srvmon.RedisAddr, srvmon.RedisPass)
	rawDB = server.SetupPostgres(checkCfg.Database.URI)
	db = rawDB

	sigtermC := make(chan os.Signal, 1)
	signal.Notify(sigtermC, os.Interrupt)

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

	monitor.ReloadTeamsAndServices(baseIP)

	// Create control ctx, to let the pg-listener goroutine be stopped on demand.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Separate thread listens for updates from the DB and automatically reloads.
	go monitor.ListenForConfigUpdatesFromPG(ctx, baseIP)

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
	go monitor.Run(&event, &srvmon)

	// Stop if: 1. The event is over 2. monitor has stopped 3. received ctrl+C (SIGTERM)
	select {
	case <-time.After(time.Until(event.End)):
		Logger.Info("Event is over. Done Checking Services")
	case <-monitor.done:
	case <-sigtermC:
	}
}
