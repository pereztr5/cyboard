package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

type store struct {
	*redis.Pool
}

func (s store) getTimeout() time.Duration {
	c := s.Get()
	defer c.Close()

	timeout, err := redis.Int64(c.Do("GET", coordination.RedisKeyTimeout))
	if err != nil {
		log.Fatalln("unable to get timeout:", err)
	}
	return time.Duration(timeout)
}

func (s store) getTeamsAndServices(teamIPs []int16, scriptsDir string) (map[int16]BlueteamView, []MonitorService, error) {
	c := s.Get()
	defer c.Close()

	teams := map[int16]BlueteamView{}
	{
		values, err := redis.ByteSlices(
			c.Do("HMGET", redis.Args{}.Add(coordination.RedisKeyTeams).AddFlat(teamIPs)...))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch all team targets: %v (targets=%v)", err, teamIPs)
		}

		for i, data := range values {
			ip := teamIPs[i]
			t := BlueteamView{}
			err = json.Unmarshal(data, &t)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal team: %v (ip=%d, data=%v)", err, ip, data)
			}
			teams[ip] = t
		}
	}

	services := []MonitorService{}
	{
		data, err := redis.Bytes(c.Do("GET", coordination.RedisKeyServices))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch services: %v", err)
		}

		err = json.Unmarshal(data, &services)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal json services: %v", err)
		}
	}

	return teams, services, nil
}

func (s store) sendResults(targetIP int16, results []ServiceCheck) error {
	c := s.Get()
	defer c.Close()

	data, err := json.Marshal(results)
	if err != nil {
		return err
	}

	_, err = c.Do("RPUSH", coordination.FmtResultsKey(targetIP), data)
	return err
}
