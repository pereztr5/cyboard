package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/meatballhat/negroni-logrus"
	"github.com/spf13/viper"
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

func SetupScoringLoggers(cfg *viper.Viper) {
	setupLogManager(cfg)

	Logger = LogManager.newLogger(applicationLogName)
	CaptFlagsLogger = LogManager.newLogger(capturedFlagsLogName)

	requestsLog := LogManager.newLogger(requestLogName)
	RequestLogger = LogManager.newRequestMiddleware(requestsLog)
}

func SetupCheckServiceLogger(cfg *viper.Viper) {
	setupLogManager(cfg)

	Logger = LogManager.newLogger(checkServiceLogName)
}

func setupLogManager(cfg *viper.Viper) {
	if LogManager == nil {
		LogManager = &LoggerManager{
			Level:     mustParseLevel(cfg.GetString("log.level")),
			UseStdout: cfg.GetBool("log.stdout"),
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
		team, teamName := "<none>", "<none>"
		if req.Context().Value("team") != nil {
			t := req.Context().Value("team").(Team)
			team = t.Group
			teamName = t.Name
		}

		return entry.WithFields(logrus.Fields{
			"request":  req.RequestURI,
			"method":   req.Method,
			"remote":   remoteAddr,
			"team":     team,
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
		fmt.Printf("Failed to parse log level '%s': %v", lvl, err)
		fmt.Println("Valid levels:", logrus.AllLevels)
		os.Exit(1)
	}
	return lvl
}
