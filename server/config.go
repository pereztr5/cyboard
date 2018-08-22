package server

import (
	"fmt"
	"time"

	"github.com/pereztr5/cyboard/server/models"
)

type LogSettings struct {
	Level  string `mapstructure:"level"`
	Stdout bool   `mapstructure:"stdout"`
}

type DBSettings struct {
	URI string `mapstructure:"postgres_uri"`
}

type ServerSettings struct {
	IP                string
	HTTPPort          string   `mapstructure:"http_port"`
	HTTPSPort         string   `mapstructure:"https_port"`
	CertPath          string   `mapstructure:"cert"`
	CertKeyPath       string   `mapstructure:"key"`
	SpecialChallenges []string `mapstructure:"special_challenges"`

	EventSettings
}

type Configuration struct {
	Appname  string      `mapstructure:"appname"`
	Log      LogSettings `mapstructure:"log"`
	Server   ServerSettings
	Database DBSettings
}

type EventSettings struct {
	ChecksDir string    `mapstructure:"checks_dir"`
	End       time.Time `mapstructure:"event_end_time"`
	Intervals time.Duration
	Timeout   time.Duration
	OnBreak   bool `mapstructure:"on_break"`
}

func (es *EventSettings) String() string {
	return fmt.Sprintf(`Event{end=%v, interval=%v, timeout=%v, OnBreak=%v}`,
		es.End.Format(time.UnixDate), es.Intervals, es.Timeout, es.OnBreak)
}

type CheckConfiguration struct {
	Event    EventSettings
	Log      LogSettings
	Database DBSettings
	Services []models.Service
}
