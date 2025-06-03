package config

import (
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

// InitLog Initialize logging settings
func InitLog(logDir string, logFilename string, lev string) {
	log.Debugf("LOG dir: %s, filename: %s, level: %s", logDir, logFilename, lev)

	// Create log directory if it doesn't exist
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Errorf("Failed to create log directory: %v", err)
		}
	}

	// Handle empty log filename
	if logFilename == "" {
		path, _ := os.Executable()
		_, exec := filepath.Split(path)
		logFilename = exec + ".log"
		log.Warning("LOG filename is empty, using executable name")
	}

	// Construct full log path
	fullLogPath := logFilename
	if logDir != "" {
		fullLogPath = filepath.Join(logDir, logFilename)
	}

	// Set to generate a logging file every day
	// Keep logs for 15 days
	writer, err := rotatelogs.New(fullLogPath+".%Y%m%d",
		rotatelogs.WithLinkName(fullLogPath),
		rotatelogs.WithRotationCount(15),
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour))

	if err != nil {
		log.Errorf("Failed to initialize log rotation: %v", err)
		return
	}

	log.SetOutput(writer)

	level, err := log.ParseLevel(lev)
	if err == nil {
		log.SetLevel(level)
	} else {
		log.Errorf("Invalid log level '%s', using default", lev)
	}
}
