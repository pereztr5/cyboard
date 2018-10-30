package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

func setupRedis() *redis.Pool {
	url, ok := os.LookupEnv("CY_REDIS_URL")
	if !ok {
		url = "redis://127.0.0.1:6379/"
	}
	pool := &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 90 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(url) },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	c := pool.Get()
	defer c.Close()
	_, err := c.Do("PING")
	if err != nil {
		log.Fatalf("setupRedis: failed to create connection pool: url=%q, err=%v", url, err)
	}

	return pool
}

func parseTargets(targets string) []int16 {
	parts := strings.Split(targets, ",")

	ips := []int16{}
	for _, p := range parts {
		i, err := strconv.ParseInt(p, 10, 16)
		if err != nil {
			log.Fatalf("unable to parse targets: ip=%q, err=%v", p, err)
		}
		ips = append(ips, int16(i))
	}
	return ips
}

func checkScriptDir(path string) string {
	// do some path look validation
	return path
}

func main() {
	log.Println("foo")
}
