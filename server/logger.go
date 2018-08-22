package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/meatballhat/negroni-logrus"
	"github.com/pereztr5/cyboard/server/models"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultRotateSize = 100 // MB
	DefaultMaxBackups = 10
	LogDir            = "log"

	applicationLogName   = "server.log"
	capturedFlagsLogName = "captured_flags.log"
	requestLogName       = "requests.log"
	checkServiceLogName  = "checks.log"
)

var (
	LogManager *LoggerManager

	Logger          *logrus.Logger            // Service logger (either for Checks, or for Scoring)
	CaptFlagsLogger *logrus.Logger            // Just logs flags as they are captured
	RequestLogger   *negronilogrus.Middleware // Sits in the server mw stack, not called directly
)

// SetupScoringLoggers uses the log configuration to instantiate
// global loggers used by the CTF + Web server component
func SetupScoringLoggers(lc *LogSettings) {
	setupLogManager(lc)

	Logger = LogManager.newLogger(applicationLogName)
	CaptFlagsLogger = LogManager.newLogger(capturedFlagsLogName)
	models.CaptFlagsLogger = CaptFlagsLogger

	requestsLog := LogManager.newLogger(requestLogName)
	RequestLogger = LogManager.newRequestMiddleware(requestsLog)
}

// SetupCheckServiceLogger uses the log configuration to instantiate
// a global logger used by the Service Checker component
func SetupCheckServiceLogger(lc *LogSettings) {
	setupLogManager(lc)

	Logger = LogManager.newLogger(checkServiceLogName)
}

func setupLogManager(lc *LogSettings) {
	if LogManager == nil {
		LogManager = &LoggerManager{
			Level:     mustParseLevel(lc.Level),
			UseStdout: lc.Stdout,
			Formatter: &logrus.TextFormatter{FullTimestamp: true},
		}
	}
}

// LoggerManager holds all the log settings together, which are used
// across the multiple loggers the server uses.
type LoggerManager struct {
	UseStdout bool
	Level     logrus.Level
	Formatter *logrus.TextFormatter

	loggers []*logrus.Logger
}

// newLogger makes a new Logrus logger with the Manager's settings
// (Eg. log level and 'to file' or 'to stdout')
func (m *LoggerManager) newLogger(filename string) *logrus.Logger {
	l := logrus.New()
	l.Out = m.newLogOutput(filename)
	l.Level = m.Level
	l.Formatter = m.Formatter
	if !m.UseStdout {
		// lumberjack handles write locks
		l.SetNoLock()
	}

	m.loggers = append(m.loggers, l)
	return l
}

// newLogOutput makes a new appropriate io.Writer (stdout, or a file)
func (m *LoggerManager) newLogOutput(logfilename string) io.WriteCloser {
	var out io.WriteCloser
	if m.UseStdout {
		out = os.Stdout
	} else {
		out = &lumberjack.Logger{
			Filename:   filepath.Join(LogDir, logfilename),
			MaxSize:    DefaultRotateSize,
			MaxBackups: DefaultMaxBackups,
			Compress:   true,
		}
	}
	return out
}

// newRequestMiddleware creates a new request logger, tailored to the scoring engine
func (m *LoggerManager) newRequestMiddleware(logger *logrus.Logger) *negronilogrus.Middleware {
	mw := negronilogrus.NewMiddlewareFromLogger(logger, "request")

	// The default "started handling request" line doesn't seem useful at this time.
	mw.SetLogStarting(false)

	// Override the fields added to log entries.
	// The removed defaults set by the library are left as comments.
	mw.Before = func(entry *logrus.Entry, req *http.Request, remoteAddr string) *logrus.Entry {
		role, teamName := "<none>", "<none>"
		t := getCtxTeam(req)
		if t != nil {
			role = t.RoleName.String()
			teamName = t.Name
		}

		return entry.WithFields(logrus.Fields{
			"request":  req.RequestURI,
			"method":   req.Method,
			"remote":   remoteAddr,
			"teamRole": role,
			"teamName": teamName,
		})
	}
	mw.After = func(entry *logrus.Entry, res negroni.ResponseWriter, latency time.Duration, name string) *logrus.Entry {
		return entry.WithFields(logrus.Fields{
			"status": res.Status(),
			//"text_status": http.StatusText(res.Status()),
			"took": latency,
			//fmt.Sprintf("measure#%s.latency", name): latency.Nanoseconds(),
		})
	}
	return mw
}

func mustParseLevel(levelStr string) logrus.Level {
	lvl, err := logrus.ParseLevel(levelStr)
	if err != nil {
		fmt.Printf("Failed to parse log level '%s': %v\n", levelStr, err)
		fmt.Println("Valid levels:", logrus.AllLevels)
		os.Exit(1)
	}
	return lvl
}
