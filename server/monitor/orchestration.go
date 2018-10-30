package monitor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"

	"github.com/pereztr5/cyboard/server/models"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

func setupRedis(connType, addr string) *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     15,
		IdleTimeout: 90 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial(connType, addr) },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	c := pool.Get()
	defer c.Close()
	_, err := c.Do("PING")
	if err != nil {
		Logger.WithFields(logrus.Fields{
			"error":  err,
			"config": fmt.Sprintf("(%s, %s)", connType, addr),
		}).Fatal("setupRedis: failed to create connection pool")
	}

	rstore = pool
	return pool
}

func redisUpdateTimeout(pool *redis.Pool, timeout time.Duration) {
	c := pool.Get()
	defer c.Close()

	c.Do("SET", coordination.RedisKeyTimeout, timeout.Nanoseconds())
}

func redisUpdateTeamsAndServices(pool *redis.Pool, teams []models.BlueteamView, services []models.MonitorService) {
	c := pool.Get()
	defer c.Close()

	// Save services
	{
		data, err := json.Marshal(services)
		if err != nil {
			Logger.WithError(err).Error("failed to marshal services for redis")
		} else {
			_, err := c.Do("SET", coordination.RedisKeyServices, data)
			if err != nil {
				Logger.WithError(err).Error("failed to set services in redis")
			}
		}
	}

	// Save teams
	{
		pack := map[int16][]byte{}
		for _, t := range teams {
			data, err := json.Marshal(t)
			if err != nil {
				Logger.WithError(err).WithField("team", t).Error("failed to marshal team for redis")
			} else {
				pack[t.BlueteamIP] = data
			}
		}
		c.Send("DEL", coordination.RedisKeyTeams)
		c.Do("HMSET", redis.Args{}.Add(coordination.RedisKeyTeams).AddFlat(pack)...)
		_, err := c.Receive()
		if err != nil {
			Logger.WithError(err).Error("failed to hmset teams in redis")
		}
	}
}

func receiveResults(statusChan chan CheckResults, pool *redis.Pool, timeoutSecs int64, timestamp *time.Time, waitKey string) {
	c := pool.Get()
	defer c.Close()

	// Wait up to the deadline for results from redis
	res := CheckResults{}
	popResponse, err := redis.ByteSlices(c.Do("BLPOP", waitKey, timeoutSecs))
	if err != nil {
		res.Err = err
		if res.Err == redis.ErrNil {
			res.Err = fmt.Errorf("Timeout reached on %q", waitKey)
		}
	} else {
		// redis BLPOP returns a pair of [<key popped from> <data>]
		data := popResponse[1]
		res.Results = []models.ServiceCheck{}
		res.Err = json.Unmarshal(data, &res.Results)

		for _, r := range res.Results {
			r.CreatedAt = *timestamp
		}
	}

	// Send (results, err) back on statusChan
	statusChan <- res
	return
}

func prepareServices(services []models.MonitorService, baseIP string) []models.MonitorService {
	// argCache saves a few cpu cycles doing the same argument variable substitution
	argCache := map[string]string{}

	for i := range services {
		srv := &services[i]

		// Substitute any {IP} arg using the 'baseIP'
		//     e.g. 192.168.5.{t}
		// Then each check agent will expand the {t} into the team's identifying IP octet
		// along with any other variable args.
		for j, arg := range srv.Args {
			s, ok := argCache[arg]
			if !ok {
				s = arg
				s = strings.Replace(arg, "{IP}", baseIP+"{t}", -1)
				s = strings.Replace(arg, "{TEAM_4TH_OCTET}", "{t}", -1)
				argCache[arg] = s
			}
			srv.Args[j] = s
		}
	}

	// Print all services from the Checks.
	if Logger.IsLevelEnabled(logrus.InfoLevel) {
		Logger.Info("All services:")
		for _, srv := range services {
			Logger.Infof(`  [%d] MonitorService{name=%q, fullcmd="%s %s"}`,
				srv.ID, srv.Name,
				filepath.Base(srv.Script), strings.Join(srv.Args, " "))
		}
	}

	return services
}
