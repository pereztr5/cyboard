package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pereztr5/cyboard/server/monitor/coordination"
)

const heartbeat = 60 * time.Second

func (s store) subscribeLoop(onMessage func(redis.Message) error) error {
	c := s.Get()
	defer c.Close()

	psc := redis.PubSubConn{Conn: c}
	if err := psc.Subscribe(coordination.RedisKeySchedule); err != nil {
		return fmt.Errorf("error subscribing: %v", err)
	}
	done := make(chan error, 1)

	go func() {
		for {
			switch msg := psc.Receive().(type) {
			case error:
				done <- msg
				return
			case redis.Message:
				err := onMessage(msg)
				if err != nil {
					log.Println("pubsub error:", err)
				}
			}
		}
	}()

	tick := time.NewTicker(heartbeat)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if psc.Ping("") != nil {
				break
			}
		case err := <-done:
			log.Println("pubsub fatal error:", err)
			break
		}
	}

	psc.Unsubscribe()
	return <-done
}
