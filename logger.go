package main

import (
	"io"

	"github.com/gruntwork-io/go-commons/logging"
	"github.com/sirupsen/logrus"
)

const DEFAULT_LOG_LEVEL = logrus.InfoLevel

// GetProjectLogger returns a logging instance for this project
func GetProjectLogger() *logrus.Entry {
	return logging.GetLogger("fetch", "")
}

// GetProjectLoggerWithWriter creates a logger around the given output stream
func GetProjectLoggerWithWriter(writer io.Writer) *logrus.Entry {
	logger := GetProjectLogger()
	logger.Logger.Out = writer
	return logger
}
