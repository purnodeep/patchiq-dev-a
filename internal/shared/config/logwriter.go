package config

import (
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogWriter returns an io.Writer for the given LogSettings.
// When log.file is empty, it writes to stdout.
// When log.file is set, it writes to a rotating file via lumberjack.
func LogWriter(cfg LogSettings) io.Writer {
	if cfg.File == "" {
		return os.Stdout
	}

	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100 // 100 MB default
	}

	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 5
	}

	maxAge := cfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 30
	}

	return &lumberjack.Logger{
		Filename:   cfg.File,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}
}
