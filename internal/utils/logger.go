package utils

import (
	"os"
	"strconv"

	"github.com/jbrodriguez/mlog"
)

func init() {
	level := mlog.LevelError

	logLevelStr := os.Getenv("LOG_LEVEL")
	switch logLevelStr {
	case "trace":
		level = mlog.LevelTrace
	case "debug":
		level = mlog.LevelTrace // mlog has no LevelDebug; Trace is the most verbose
	case "info":
		level = mlog.LevelInfo
	case "warn", "warning":
		level = mlog.LevelWarn
	case "error":
		level = mlog.LevelError
	case "none", "off":
		level = 0 // suppress everything
	default:
		// Try numeric
		if n, err := strconv.Atoi(logLevelStr); err == nil {
			level = mlog.LogLevel(n)
		}
	}

	mlog.Start(level, "")
}
