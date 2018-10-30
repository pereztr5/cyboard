package monitor

import (
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"

	"github.com/pereztr5/cyboard/server"
	"github.com/pereztr5/cyboard/server/models"
)

// Utilize type aliases while migrating monitor to separate package

type ScheduledBreak = server.ScheduledBreak
type Configuration = server.Configuration
type EventSettings = server.EventSettings
type ServiceMonitorSettings = server.ServiceMonitorSettings

// Utilize globals while migrating monitor to separate package
var (
	Logger *logrus.Logger
	rawDB  *pgx.ConnPool
	db     models.DBClient

	rstore *redis.Pool
)
