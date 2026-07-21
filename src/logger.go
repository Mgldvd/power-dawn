package main

import (
	"io"
	"os"
	"time"

	clog "github.com/charmbracelet/log"
)

const appLogPath = "/var/log/power-dawn.log"

// appLogger is the package-level structured logger.
// It is initialised by initLogger() in main() after the root check.
var appLogger *clog.Logger

// initLogger opens (or creates) the app log file and configures the logger.
// Must be called after the root check so we are guaranteed write permission.
func initLogger() {
	f, err := os.OpenFile(appLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		// Unexpected under root; fall back to stderr and keep going.
		appLogger = clog.NewWithOptions(os.Stderr, clog.Options{
			ReportTimestamp: true,
			TimeFormat:      time.DateTime,
		})
		appLogger.Error("could not open app log file", "path", appLogPath, "err", err)
		return
	}

	appLogger = clog.NewWithOptions(f, clog.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateTime,
		Level:           clog.DebugLevel,
	})
}

// logOrDiscard returns a no-op writer if appLogger is not yet set up.
// Used as a safety guard for any code that might run before initLogger.
func logOrDiscard() *clog.Logger {
	if appLogger == nil {
		return clog.NewWithOptions(io.Discard, clog.Options{})
	}
	return appLogger
}
