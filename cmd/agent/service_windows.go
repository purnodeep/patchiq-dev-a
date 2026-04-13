//go:build windows

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/shared/config"
	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
)

func isWindowsService() bool {
	return agent.IsWindowsService()
}

func runAsWindowsService(configPath string) {
	cfg := loadConfig(configPath)
	logLevel := parseLogLevel(cfg.log.Level)
	logger := slog.New(piqotel.NewHandler(config.LogWriter(cfg.log), &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	if err := agent.RunAsService(logger, func(ctx context.Context) error {
		cancel := context.CancelFunc(func() {}) // service manages cancellation via svc.Handler
		return runDaemon(ctx, cancel, configPath)
	}); err != nil {
		logger.Error("windows service failed", "error", err)
		os.Exit(1)
	}
}
