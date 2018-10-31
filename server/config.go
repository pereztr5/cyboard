package server

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

type Configuration struct {
	Database       DBSettings
	Log            LogSettings
	Event          EventSettings
	Server         ServerSettings
	ServiceMonitor ServiceMonitorSettings `mapstructure:"service_monitor"`
}

type ScheduledBreak struct {
	StartsAt time.Time     `mapstructure:"at"`
	GoesFor  time.Duration `mapstructure:"for"`
}

func (sb *ScheduledBreak) End() time.Time {
	return sb.StartsAt.Add(sb.GoesFor)
}

func (sb ScheduledBreak) String() string {
	return fmt.Sprintf(`ScheduledBreak{at=%v, for=%v}`,
		sb.StartsAt.Format(time.Stamp), sb.GoesFor)
}

type DBSettings struct {
	URI string `mapstructure:"postgres_uri"`
}

type EventSettings struct {
	Start  time.Time
	End    time.Time
	Breaks []ScheduledBreak
	// OnBreak bool `mapstructure:"on_break"`
}

func (es EventSettings) String() string {
	return fmt.Sprintf(
		`Event{start=%v, end=%v, breaks=%v}`,
		es.Start.Format(time.Stamp), es.End.Format(time.Stamp), es.Breaks)
}

type LogSettings struct {
	Level  string `mapstructure:"level"`
	Stdout bool   `mapstructure:"stdout"`
}

type ServerSettings struct {
	Appname     string `mapstructure:"appname"`
	IP          string
	HTTPPort    string `mapstructure:"http_port"`
	HTTPSPort   string `mapstructure:"https_port"`
	CertPath    string `mapstructure:"cert"`
	CertKeyPath string `mapstructure:"key"`

	Compress   bool
	RateLimit  bool   `mapstructure:"rate_limit"`
	CtfFileDir string `mapstructure:"ctf_file_dir"`
}

type ServiceMonitorSettings struct {
	RedisConnType string `mapstructure:"redis_conn_type"`
	RedisAddr     string `mapstructure:"redis_addr"`
	RedisPass     string `mapstructure:"redis_pass"`

	BaseIP    string `mapstructure:"base_ip_prefix"`
	Intervals time.Duration
	Timeout   time.Duration

	ChecksDir string `mapstructure:"checks_dir"`
}

// Validate checks for constraints on the config, including: Event start is after event end,
// negative times (interval, timeout), breaks out of order, overlapping breaks,
// break occurs before/after event starts/ends, and base_ip is a 3-octet IP prefix.
func (cfg *Configuration) Validate() error {
	event, mon := cfg.Event, cfg.ServiceMonitor

	if event.Start.After(event.End) {
		return fmt.Errorf("Event starts after it ends: event=%v", &event)
	}

	if mon.Intervals < 1 {
		return fmt.Errorf("Check interval must be positive: service_monitor.intervals=%v",
			mon.Intervals)
	} else if mon.Timeout < 1 {
		return fmt.Errorf("Timeout must be positive: service_monitor.timeout=%v",
			mon.Timeout)
	}

	a := event.Breaks
	b := append([]ScheduledBreak(nil), a...)
	sort.Slice(b, func(i, j int) bool { return b[i].StartsAt.Before(b[j].StartsAt) })
	for i := 0; i < len(a); i++ {
		if !a[i].StartsAt.Equal(b[i].StartsAt) {
			return fmt.Errorf("Breaks must be ordered earliest to latest: event.breaks=%v",
				event.Breaks)
		}

		brk := a[i]
		if brk.GoesFor < 1 {
			return fmt.Errorf("Breaks must go for a positive amount of time: "+
				"event.break[%d]=%v", i, brk)
		}

		if brk.StartsAt.Before(event.Start) {
			return fmt.Errorf("Breaks must start after the event has started: "+
				"event.break[%d]=%v, event.start=%v",
				i, brk, event.Start)
		} else if brk.End().After(event.End) {
			return fmt.Errorf("Breaks must end before the event has ended: "+
				"event.break[%d]=%v (ends_at=%v), event.end=%v",
				i, brk, brk.End().Format(time.Stamp), event.End)
		}
	}

	for i := 0; i < len(event.Breaks)-1; i++ {
		brk, next := event.Breaks[i], event.Breaks[i+1]
		if brk.End().After(next.StartsAt) {
			return fmt.Errorf("Breaks must not overlap: break[%d]=%v, break[%d]=%v",
				i, brk, i+1, next)
		}
	}

	if !IPish(mon.BaseIP) {
		return fmt.Errorf("3 octet IP Prefix should have the form \"192.168.0.\" "+
			"but got the following instead: event.base_ip_prefix=%q", mon.BaseIP)
	}

	return nil
}

// IPish tests whether a string looks like the first 3 octets of an IPv4 address.
func IPish(ip_prefix string) bool {
	test_ip := ip_prefix + "0"
	return strings.HasSuffix(ip_prefix, ".") && (net.ParseIP(test_ip) != nil)
}
