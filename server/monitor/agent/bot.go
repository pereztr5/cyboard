package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

const timeoutAdj = 100 * time.Millisecond

type bleepbloop struct {
	targets    []int16
	scriptsDir string
	timeout    time.Duration

	teams    map[int16]BlueteamView
	services []MonitorService
	checks   [][]Check
}

func newBleepBloop(targets, scriptsDir string) *bleepbloop {
	return &bleepbloop{
		targets:    parseTargets(targets),
		scriptsDir: checkScriptDir(scriptsDir),
	}
}

func (bot *bleepbloop) reload(s store) error {
	var err error
	bot.timeout, err = s.getTimeout()
	if err != nil {
		return err
	}

	bot.teams, bot.services, err = s.getTeamsAndServices(bot.targets, bot.scriptsDir)
	if err != nil {
		return err
	}

	bot.checks = make([][]Check, 0, len(bot.teams))
	for i := range bot.teams {
		team := bot.teams[i]
		teamChecks, err := prepareChecks(bot.services, &team, bot.scriptsDir)
		if err != nil {
			return err
		}
		bot.checks = append(bot.checks, teamChecks)
	}
	return nil
}

func (bot *bleepbloop) runChecks(s store, teamIP int16, teamChecks []Check) {
	results := make([]ServiceCheck, 0, len(teamChecks))
	resultsChan := make(chan ServiceCheck)

	for i := range teamChecks {
		check := teamChecks[i]
		go func(chk *Check) { resultsChan <- runCmd(chk, bot.timeout) }(&check)
	}

	deadline := time.After(bot.timeout + timeoutAdj)

	// collect results and publish
	for i := 0; i < len(teamChecks); i++ {
		select {
		case res := <-resultsChan:
			results = append(results, res)
		case <-deadline:
			// goroutines not responding; should never happen
			log.Fatalf("goroutine failed to ack back in time (teamIP=%d)", teamIP)
		}
	}

	rs := []string{}
	for _, res := range results {
		rs = append(rs, fmt.Sprintf("%d:%v", res.ServiceID, res.Status))
	}
	log.Printf("results (ip=%v): %v", teamIP, rs)

	if err := s.sendResults(teamIP, results); err != nil {
		log.Println("send results failed: %v (teamIP=%d)", err, teamIP)
	}
}

func (bot *bleepbloop) messageFunc(s store) func(redis.Message) error {
	fn := func(msg redis.Message) error {
		if len(msg.Data) < 1 {
			return fmt.Errorf("empty message")
		}

		var doReload bool = bot.teams == nil

		switch string(msg.Data) {
		case coordination.SignalRun:
		case coordination.SignalReloadThenRun:
			doReload = true
		default:
			return fmt.Errorf("unknown command signal: %s", msg.Data)
		}

		// Refresh data as needed (on command from master or when no data available)
		if doReload {
			if err := bot.reload(s); err != nil {
				return fmt.Errorf("reload teams & services failed: %v", err)
			}
		}

		// Run commands against all the target teams
		wg := sync.WaitGroup{}
		wg.Add(len(bot.teams))
		var idx int
		for teamIP := range bot.teams {
			go func(ip int16, checks []Check) {
				bot.runChecks(s, ip, checks)
				wg.Done()
			}(teamIP, bot.checks[idx])
			idx++
		}
		wg.Wait()
		return nil
	}

	return fn
}

func (bot *bleepbloop) run(s store) {
	log.Printf("b1e3pBl00p -=-=- daemonizing -=-=- targets=%v", bot.targets)
	err := s.subscribeLoop(bot.messageFunc(s))
	if err != nil {
		log.Println("pubsub fatal error:", err)
	}
}
