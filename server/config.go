package server

import (
	"fmt"
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

func (sb *ScheduledBreak) String() string {
	return fmt.Sprintf(`ScheduledBreak{at=%v, for=%v}`,
		sb.StartsAt.Format(time.UnixDate), sb.GoesFor)
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

func (es *EventSettings) String() string {
	return fmt.Sprintf(
		`Event{start=%v, end=%v, breaks=%v}`,
		es.Start.Format(time.UnixDate), es.End.Format(time.UnixDate), es.Breaks)
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
}

type ServiceMonitorSettings struct {
	ChecksDir string `mapstructure:"checks_dir"`
	Intervals time.Duration
	Timeout   time.Duration
}
