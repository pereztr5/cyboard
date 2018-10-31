package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

func envElse(key, other string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		val = other
	}
	return val
}

func setupRedis(url string) *redis.Pool {
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
			log.Fatalf("unable to parse targets: targets=%v, ip=%q, err=%v", parts, p, err)
		}
		ips = append(ips, int16(i))
	}
	return ips
}

func checkScriptDir(path string) string {
	cleaned, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("checkScriptDir abspath lookup failed: %v (path=%q)", err, path)
	}

	stat, err := os.Stat(cleaned)
	if err != nil {
		log.Fatalf("checkScriptDir error: %v (path=%q)", err, path)
	} else if !stat.IsDir() {
		log.Fatalf("checkScriptDir error: path is not a directory (path=%q)", path)
	}
	return cleaned
}
