package monitor

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus logger
type Logger struct {
	*logrus.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level, output string) *Logger {
	logger := logrus.New()

	// Set log level
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set output
	var writers []io.Writer
	switch output {
	case "file":
		file, err := os.OpenFile("logs/trading.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Warnf("Failed to open log file: %v, falling back to console", err)
			writers = []io.Writer{os.Stdout}
		} else {
			writers = []io.Writer{file}
		}
	case "both":
		file, err := os.OpenFile("logs/trading.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			writers = []io.Writer{os.Stdout}
		} else {
			writers = []io.Writer{os.Stdout, file}
		}
	default:
		writers = []io.Writer{os.Stdout}
	}

	logger.SetOutput(io.MultiWriter(writers...))
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return &Logger{Logger: logger}
}

