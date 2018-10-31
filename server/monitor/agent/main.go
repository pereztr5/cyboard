package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	PROG  string = "agent"
	BUILD string
)

const USAGE = `%s  %s
Scoring script execution daemon - Runs commands and reports results back

USAGE:
  %s

Environment Vars:
  CY_REDIS_URL    Connection string for redis. [DEFAULT: redis://127.0.0.1:6379]
                  example: redis://user:secret@127.0.0.1:6379/0
  CY_SCRIPTS_DIR  Path to scripts folder [DEFAULT: "./scripts"]
  CY_TARGETS      Comma-separated list of {t} network identifier IPs to target
                  example: "1,3,10" becomes "192.168.0.1", "192.168.0.3", ...
`

func main() {
	flag.Usage = func() { fmt.Fprintf(os.Stderr, USAGE, PROG, BUILD, os.Args[0]) }
	flag.Parse()

	url := envElse("CY_REDIS_URL", "redis://127.0.0.1:6379/")
	rstore := store{setupRedis(url)}
	defer rstore.Close()

	scriptsDir := envElse("CY_SCRIPTS_DIR", "./scripts")
	targets := os.Getenv("CY_TARGETS")
	if targets == "" {
		log.Fatal("no targets specified (see --help)")
	}

	bot := newBleepBloop(targets, scriptsDir)
	bot.run(rstore)
}
