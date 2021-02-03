package main

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

const DEFAULT_LOG_LEVEL = logrus.InfoLevel

// CreateLogger creates a logger. If debug is set, we use ErrorLevel to enable verbose output, otherwise - only errors are shown
func CreateLogger(lvl logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(lvl)
	logger.SetOutput(os.Stderr) // Fetch should output all it's logs to stderr by default
	logger.SetFormatter(&Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "%time% [%lvl%] [%prefix%] %msg%",
	})

	return logger
}

// CreateLogEntry creates a logger entry with the given prefix field
func CreateLogEntry(prefix string, level logrus.Level) *logrus.Entry {
	logger := CreateLogger(level)
	var fields logrus.Fields
	if prefix != "" {
		fields = logrus.Fields{"prefix": prefix}
	} else {
		fields = logrus.Fields{"prefix": "fetch"}
	}
	return logger.WithFields(fields)
}

// CreateLogEntryWithWriter Create a logger around the given output stream and prefix
func CreateLogEntryWithWriter(writer io.Writer, prefix string, level logrus.Level) *logrus.Entry {
	if prefix != "" {
		prefix = fmt.Sprintf("%s", prefix)
	} else {
		prefix = fmt.Sprintf("fetch%s", prefix)
	}
	logger := CreateLogEntry(prefix, level)
	logger.Logger.SetOutput(writer)
	return logger
}
