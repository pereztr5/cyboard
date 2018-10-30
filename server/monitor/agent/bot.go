package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

type bleepbloop struct {
	targets    []int16
	scriptsDir string
	timeout    time.Duration

	teams    map[int16]BlueteamView
	services []MonitorService
	checks   [][]Check
}

func (bot *bleepbloop) reload(s store) error {
	var err error
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
		chk := teamChecks[i]
		go func() { resultsChan <- runCmd(&chk, bot.timeout) }()
	}

	// collect results and publish
	for res := range resultsChan {
		results = append(results, res)
	}

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

		sig := msg.Data[0]
		switch sig {
		case coordination.SignalRun:
		case coordination.SignalReloadThenRun:
			doReload = true
		default:
			return fmt.Errorf("unknown command signal: %v", sig)
		}

		// Refresh data as needed (on command from master or when no data available)
		if doReload {
			if err := bot.reload(s); err != nil {
				return fmt.Errorf("reload teams & services failed: %v", err)
			}
		}

		// Run commands against all the target teams
		for i, team := range bot.teams {
			bot.runChecks(s, team.BlueteamIP, bot.checks[i])
		}
		return nil
	}

	return fn
}

func (bot *bleepbloop) run() {
	s := store{setupRedis()}
	bot.timeout = s.getTimeout()
	bot.targets = parseTargets(os.Getenv("CY_TARGETS"))
	bot.scriptsDir = checkScriptDir(os.Getenv("CY_SCRIPTS_DIR"))

	err := s.subscribeLoop(bot.messageFunc(s))
	if err != nil {
		log.Print(err)
	}
}
