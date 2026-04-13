package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/skenzeriq/patchiq/cmd/agent/cli"
	"github.com/skenzeriq/patchiq/cmd/agent/sysinfo"
	"github.com/skenzeriq/patchiq/internal/agent"
	agentapi "github.com/skenzeriq/patchiq/internal/agent/api"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"github.com/skenzeriq/patchiq/internal/agent/executor"
	"github.com/skenzeriq/patchiq/internal/agent/inventory"
	"github.com/skenzeriq/patchiq/internal/agent/patcher"
	"github.com/skenzeriq/patchiq/internal/agent/store"
	"github.com/skenzeriq/patchiq/internal/agent/system"
	"github.com/skenzeriq/patchiq/internal/shared/config"
	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
)

// Set by goreleaser via -ldflags.
var (
	version = "dev"
	commit  = "unknown"
)

// sqlOpen is the function used to open a database connection for isEnrolled.
// It is a variable so tests can override it.
var sqlOpen = sql.Open

func main() {
	// Windows self-elevation: when double-clicked (no args) without admin
	// rights, re-invoke this exe with ShellExecute("runas") so Windows shows
	// the UAC prompt. This makes the double-click-and-paste-token flow work
	// without the operator opening PowerShell manually. Only kicks in on
	// Windows and only when no subcommand was passed (avoids double UAC on
	// explicit `patchiq-agent install ...` invocations).
	if runtime.GOOS == "windows" && len(os.Args) == 1 && !cli.IsAdmin() {
		if err := cli.RelaunchAsAdmin(); err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to elevate: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0) // parent exits; elevated child is now running
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			os.Exit(cli.RunInstall(os.Args[2:]))
		case "status":
			os.Exit(cli.RunStatus(os.Args[2:]))
		case "scan":
			os.Exit(cli.RunScan(os.Args[2:]))
		case "service":
			os.Exit(cli.RunService(os.Args[2:]))
		case "--help", "-h", "help":
			cli.Usage()
			os.Exit(0)
		}
	}

	// Parse --config flag from os.Args for the daemon path.
	// CLI subcommands above have their own config loading.
	configPath := parseConfigFlag(os.Args[1:])

	// On Linux with a display and zenity, route to GUI install if not yet
	// enrolled, or show "already enrolled" dialog if double-clicked after
	// setup. This MUST come before the generic first-run fallback below,
	// otherwise fresh installs (no config file) fall through to the TUI
	// which silently fails when Terminal=false in the .desktop launcher.
	if len(os.Args) == 1 && runtime.GOOS == "linux" && os.Getenv("DISPLAY") != "" && cli.HasZenity() {
		if !configFileExists(configPath) {
			os.Exit(cli.RunGUIInstall(nil))
		}
		cfg := loadConfig("")
		if !isEnrolled(cfg.dataDir, cfg.dbFile) {
			os.Exit(cli.RunGUIInstall(nil))
		}
		os.Exit(cli.ShowAlreadyEnrolledDialog())
	}

	// First-run auto-launch: if invoked with no subcommand and no config file
	// exists, run the install wizard. On Windows, launch the GUI wizard
	// (PowerShell WPF dialogs); on other platforms, fall back to TUI.
	if len(os.Args) == 1 && !configFileExists(configPath) {
		if runtime.GOOS == "windows" {
			os.Exit(cli.RunGUIInstall(nil))
		}
		os.Exit(cli.RunInstall([]string{"--reset-config"}))
	}

	// Check if running as Windows service
	if isWindowsService() {
		runAsWindowsService(configPath)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := runDaemon(ctx, cancel, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(ctx context.Context, cancel context.CancelFunc, configPath string) error {
	// Config
	cfg := loadConfig(configPath)

	// Logger (must come first so OTel init errors are properly structured)
	// Use LevelVar so the log level can be changed at runtime via settings.
	var logLevelVar slog.LevelVar
	logLevelVar.Set(parseLogLevel(cfg.log.Level))
	logger := slog.New(piqotel.NewHandler(config.LogWriter(cfg.log), &slog.HandlerOptions{Level: &logLevelVar}))
	slog.SetDefault(logger)

	// OTel (empty endpoint = noop by default, agent is offline-first)
	otelEndpoint := envOrDefault("PATCHIQ_AGENT_OTEL_ENDPOINT", "")
	otelShutdown, err := piqotel.Init(context.Background(), piqotel.Config{
		ServiceName:    "patchiq-agent",
		ServiceVersion: version,
		OTLPEndpoint:   otelEndpoint,
		Insecure:       true,
	})
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}

	logger.Info("patchiq-agent", "version", version, "commit", commit)
	logger.Info("starting patchiq agent", "data_dir", cfg.dataDir, "server", cfg.serverAddr)

	// SQLite
	dbPath := filepath.Join(cfg.dataDir, cfg.dbFile)
	db, err := comms.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open database %s: %w", dbPath, err)
	}
	defer db.Close()
	logger.Info("database opened", "path", dbPath)

	// Store schema (patches, history, logs tables)
	if err := store.ApplySchema(db); err != nil {
		return fmt.Errorf("apply store schema: %w", err)
	}

	// Idempotent column migrations for new UI fields
	if err := store.ApplyMigrations(db); err != nil {
		return fmt.Errorf("apply store migrations: %w", err)
	}

	// Clear seed data — real operations will populate these
	clearSeedData(ctx, db, logger)

	// Seed development data if requested
	if os.Getenv("PATCHIQ_AGENT_SEED") == "true" {
		if err := store.Seed(db); err != nil {
			return fmt.Errorf("seed database: %w", err)
		}
		logger.Info("database seeded with development data")
	}

	// Outbox, Inbox, State
	outbox := comms.NewOutbox(db)
	inbox := comms.NewInbox(db)
	state := comms.NewAgentState(db)

	// Check for pending post-reboot scan flag.
	if val, err := state.Get(ctx, "reboot_pending_scan"); err == nil && val == "true" {
		logger.Info("post-reboot scan: detected pending scan after reboot")
		if err := state.Set(ctx, "reboot_pending_scan", ""); err != nil {
			logger.Warn("clear reboot_pending_scan flag", "error", err)
		}
	}

	// Settings watcher: reads settings from SQLite periodically and caches in memory.
	// Created early so other components can reference its getters.
	settingsStore := store.NewSettingsStore(db)
	watcher := agent.NewSettingsWatcher(settingsStore, logger)
	watcher.SetLogLevelVar(&logLevelVar)

	// Module registry
	invModule := inventory.New()
	registry := agent.NewRegistry(logger)
	registry.Register(invModule)
	registry.Register(patcher.NewWithMaxConcurrentFunc(watcher.MaxConcurrentInstalls))
	registry.Register(system.New())
	registry.Register(executor.New())

	deps := agent.ModuleDeps{
		Logger:  logger,
		LocalDB: db,
		Outbox:  outbox,
		ConfigProvider: &mapConfigProvider{values: map[string]string{
			"server.http_url": cfg.serverHTTPURL,
			"data_dir":        cfg.dataDir,
		}},
		EventEmitter: agent.NoopEventEmitter{},
		FileCache:    agent.NoopFileCache{},
	}

	if err := registry.InitAll(ctx, deps); err != nil {
		return fmt.Errorf("init modules: %w", err)
	}

	// gRPC connection
	reconnCfg := comms.ReconnectConfig{
		InitialDelay: cfg.reconnInitial,
		MaxDelay:     cfg.reconnMax,
		Multiplier:   cfg.reconnMult,
		JitterFactor: cfg.reconnJitter,
	}
	client := comms.NewClient(cfg.serverAddr, reconnCfg, logger)

	if cfg.enrollToken == "" {
		logger.Warn("PATCHIQ_AGENT_ENROLLMENT_TOKEN is not set; enrollment will likely fail")
	}

	// Start modules
	if err := registry.StartAll(ctx); err != nil {
		return fmt.Errorf("start modules: %w", err)
	}

	go watcher.Start(ctx)

	// Log retention: periodically deletes old logs, outbox items, and history.
	go store.RunRetention(ctx, db, watcher.LogRetentionDays, logger)

	// Log store: created early so collection runner and command processor can write operational logs.
	logStore := store.NewLogStore(db)

	// Collection runner: periodically calls Collect() on each module and writes to outbox.
	collectionRunner := agent.NewCollectionRunner(registry.Modules(), outbox, logger)
	collectionRunner.SetIntervalFunc(watcher.ScanInterval)
	collectionRunner.SetLogWriter(logStore)
	collectionRunner.SetCacheSaver(func(ctx context.Context, data []byte) error {
		return store.SaveInventoryCache(ctx, db, data)
	})

	// Give the inventory module direct access to the outbox and cache so
	// on-demand run_scan commands use the same write path as periodic scans.
	invModule.SetOutbox(outbox)
	invModule.SetCacheSaver(func(ctx context.Context, data []byte) error {
		return store.SaveInventoryCache(ctx, db, data)
	})

	go collectionRunner.Run(ctx)

	// Status provider: created early so SetLastHeartbeat can be wired into
	// the heartbeat config callback.
	statusProvider := store.NewStatusProvider(state, version, cfg.serverAddr, db)
	statusProvider.SetInventoryHealth(invModule)

	// Command processor: polls the inbox and dispatches commands to modules.
	// Created before the gRPC goroutine so its Trigger can be wired into
	// RunConfig.OnCommandsPending (called after each inbox fetch).
	historyStore := store.NewHistoryStore(db)

	cmdProcessor := agent.NewCommandProcessor(inbox, outbox, registry, logger)
	cmdProcessor.SetOfflineCheck(watcher.IsOffline)
	cmdProcessor.SetHistoryWriter(historyStore)
	cmdProcessor.SetLogWriter(logStore)
	go cmdProcessor.Run(ctx)

	// Run enrollment + heartbeat lifecycle in background.
	// OnCommandsPending triggers the command processor after each inbox fetch.
	go func() {
		runCfg := comms.RunConfig{
			DataDir: cfg.dataDir,
			Token:   cfg.enrollToken,
			Meta: comms.AgentMeta{
				AgentVersion:    version,
				ProtocolVersion: 1,
				Capabilities:    registry.Capabilities(),
			},
			Endpoint: sysinfo.BuildEndpointInfo(logger),
			HeartbeatConfig: comms.HeartbeatConfig{
				Interval:        cfg.heartbeatInterval,
				StartTime:       time.Now(),
				Logger:          logger,
				IntervalFunc:    watcher.HeartbeatInterval,
				OfflineCheck:    watcher.IsOffline,
				OnHeartbeatSent: buildHeartbeatSentCallback(statusProvider, logStore, logger),
			},
			Outbox: outbox,
			Inbox:  inbox,
			SyncConfig: comms.SyncConfig{
				OfflineCheck:  watcher.IsOffline,
				BandwidthFunc: watcher.BandwidthLimitKbps,
			},
			OnCommandsPending: cmdProcessor.Trigger,
		}
		if err := client.Run(ctx, state, runCfg); err != nil {
			if errors.Is(err, comms.ErrShutdownRequested) {
				logger.Info("shutdown requested by server")
			} else {
				logger.Error("agent connection lifecycle failed, shutting down", "error", err)
			}
			cancel()
		}
	}()

	// HTTP API server
	patchStore := store.NewPatchStore(db)

	// Write startup log entry
	if err := logStore.WriteLog(ctx, "info", "Agent started, version "+version, "agent"); err != nil {
		logger.Warn("write startup log", "error", err)
	}
	if err := logStore.WriteLog(ctx, "info", "Modules initialized: inventory, patcher, system, executor", "agent"); err != nil {
		logger.Warn("write module init log", "error", err)
	}

	settingsProvider := agentapi.NewDynamicSettingsProvider(agentapi.SettingsInfo{
		AgentVersion:          version,
		ConfigFile:            filepath.Join(cfg.dataDir, "agent.yaml"),
		DataDir:               cfg.dataDir,
		LogFile:               cfg.log.File,
		DBPath:                dbPath,
		ServerURL:             cfg.serverAddr,
		HTTPAddr:              cfg.httpAddr,
		ScanInterval:          "6h", // TODO(PIQ-248): read from config when scan config is added
		ScanTimeout:           "300s",
		LogLevel:              cfg.log.Level,
		AutoDeploy:            false,
		HeartbeatInterval:     "30s",
		BandwidthLimitKbps:    0,
		MaxConcurrentInstalls: 1,
		ProxyURL:              "",
		AutoRebootWindow:      "",
		LogRetentionDays:      30,
		OfflineMode:           false,
	}, settingsStore)

	router := agentapi.NewRouter(agentapi.HandlerDeps{
		Status:   statusProvider,
		Patches:  patchStore,
		History:  historyStore,
		Logs:     logStore,
		Settings: settingsProvider,
		Hardware: agentapi.NewHardwareAdapter(logger),
		Software: agentapi.NewSoftwareAdapter(invModule, func(ctx context.Context) ([]byte, error) {
			cached, _, err := store.LoadInventoryCache(ctx, db)
			return cached, err
		}),
		Services:       agentapi.NewServicesAdapter(logger),
		Metrics:        &metricsAdapter{},
		SettingsUpdate: settingsStore,
		LogWriter:      logStore,
		ScanTrigger:    collectionRunner,
		APIKey:         cfg.apiKey,
	})

	httpServer := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", cfg.httpAddr)
	if err != nil {
		return fmt.Errorf("listen http %s: %w", cfg.httpAddr, err)
	}
	if err := logStore.WriteLog(ctx, "info", "HTTP server listening on "+cfg.httpAddr, "agent"); err != nil {
		logger.Warn("write http listen log", "error", err)
	}

	go func() {
		certFile := os.Getenv("PATCHIQ_AGENT_TLS_CERT")
		keyFile := os.Getenv("PATCHIQ_AGENT_TLS_KEY")
		if certFile != "" && keyFile != "" {
			logger.Info("starting HTTPS API server", "addr", cfg.httpAddr, "cert", certFile)
			if serveErr := httpServer.ServeTLS(ln, certFile, keyFile); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				logger.Error("https server error", "error", serveErr)
				cancel()
			}
		} else {
			logger.Info("starting HTTP API server", "addr", cfg.httpAddr)
			if serveErr := httpServer.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				logger.Error("http server error", "error", serveErr)
				cancel()
			}
		}
	}()

	logger.Info("agent running", "modules", len(registry.Modules()))

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down")

	// Ordered shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown", "error", err)
	}
	if err := registry.StopAll(shutdownCtx); err != nil {
		logger.Error("stop modules", "error", err)
	}
	if shutdownErr := otelShutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("otel shutdown error", "error", shutdownErr)
	}
	if err := client.Close(); err != nil {
		logger.Error("close grpc", "error", err)
	}

	logger.Info("agent stopped")
	return nil
}

type agentConfig struct {
	dataDir           string
	dbFile            string
	serverAddr        string
	serverHTTPURL     string
	httpAddr          string
	log               config.LogSettings
	reconnInitial     time.Duration
	reconnMax         time.Duration
	reconnMult        float64
	reconnJitter      float64
	enrollToken       string
	heartbeatInterval time.Duration
	apiKey            string
}

// configFileExists reports whether a config file exists at either the primary
// resolved path OR the platform-specific fallback data dir. Used by the no-args
// first-run logic to decide between launching the install wizard and starting
// the daemon. Checks both paths because on Windows a non-elevated prior run
// may have written to %UserHome%\.patchiq instead of C:\ProgramData\PatchIQ.
func configFileExists(configPath string) bool {
	if configPath == "" {
		configPath = cli.DefaultConfigPath()
	}
	if _, err := os.Stat(configPath); err == nil {
		return true
	}
	fallback := filepath.Join(cli.DefaultDataDir(), "agent.yaml")
	if fallback != configPath {
		if _, err := os.Stat(fallback); err == nil {
			return true
		}
	}
	return false
}

// parseConfigFlag extracts the --config value from args without interfering with
// subcommand parsing. Returns empty string if not found.
func parseConfigFlag(args []string) string {
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

func loadConfig(configPath string) agentConfig {
	// Load config via koanf: defaults → YAML file → env vars.
	koanfCfg, err := cli.LoadAgentConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: load config: %v\n", err)
		os.Exit(1)
	}

	// Map koanf-loaded fields into agentConfig, then fill remaining fields
	// that are not part of AgentConfig with env var fallbacks.
	return agentConfig{
		dataDir:       koanfCfg.DataDir,
		dbFile:        envOrDefault("PATCHIQ_AGENT_DB_FILE", "agent.db"),
		serverAddr:    koanfCfg.ServerAddress,
		serverHTTPURL: koanfCfg.ServerHTTPURL,
		httpAddr:      envOrDefault("PATCHIQ_AGENT_HTTP_ADDR", "127.0.0.1:8090"),
		log: config.LogSettings{
			Level:      koanfCfg.LogLevel,
			File:       envOrDefault("PATCHIQ_AGENT_LOG_FILE", ""),
			MaxSizeMB:  envOrDefaultInt("PATCHIQ_AGENT_LOG_MAX_SIZE_MB", 0),
			MaxBackups: envOrDefaultInt("PATCHIQ_AGENT_LOG_MAX_BACKUPS", 0),
			MaxAgeDays: envOrDefaultInt("PATCHIQ_AGENT_LOG_MAX_AGE_DAYS", 0),
		},
		reconnInitial:     1 * time.Second,
		reconnMax:         5 * time.Minute,
		reconnMult:        2.0,
		reconnJitter:      0.2,
		enrollToken:       envOrDefault("PATCHIQ_AGENT_ENROLLMENT_TOKEN", ""),
		heartbeatInterval: 30 * time.Second,
		apiKey:            envOrDefault("PATCHIQ_AGENT_API_KEY", ""),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// metricsAdapter implements agentapi.MetricsProvider by calling inventory.CollectMetrics.
type metricsAdapter struct{}

func (m *metricsAdapter) Metrics() (any, error) {
	return inventory.CollectMetrics(context.Background())
}

// buildHeartbeatSentCallback returns a callback that updates the status provider
// and periodically logs heartbeat events to the persistent log store. Heartbeats
// are logged at most once every 5 minutes to avoid flooding the log table.
func buildHeartbeatSentCallback(statusProvider *store.StatusProvider, logStore *store.LogStore, logger *slog.Logger) func(time.Time) {
	var lastLogged time.Time
	const logInterval = 5 * time.Minute
	return func(t time.Time) {
		statusProvider.SetLastHeartbeat(t)
		if time.Since(lastLogged) >= logInterval {
			lastLogged = t
			if err := logStore.WriteLog(context.Background(), "info", "Heartbeat sent to server", "heartbeat"); err != nil {
				logger.Warn("write heartbeat log", "error", err)
			}
		}
	}
}

// isEnrolled checks whether the agent SQLite database at dataDir/dbFile contains
// a non-empty agent_id in the agent_state table. Returns false on any error
// (missing file, missing table, empty value, open error).
func isEnrolled(dataDir, dbFile string) bool {
	dbPath := filepath.Join(dataDir, dbFile)

	db, err := sqlOpen("sqlite", dbPath+"?mode=ro")
	if err != nil {
		slog.Debug("isEnrolled: open db failed", "path", dbPath, "error", err)
		return false
	}
	defer db.Close()

	var value string
	err = db.QueryRow("SELECT value FROM agent_state WHERE key = 'agent_id'").Scan(&value)
	if err != nil {
		slog.Debug("isEnrolled: query agent_id failed", "error", err)
		return false
	}
	return value != ""
}

// clearSeedData removes development seed rows without affecting production data.
// Seed IDs use the format l0XX for logs and h0XX for history (where XX is 00-99).
// The GLOB pattern is intentionally broader than current seed data (l001-l020,
// h001-h010) to remain safe if seed rows are added later.
// Production IDs use "log-..." (from generateLogID) and other formats.
// mapConfigProvider implements agent.ConfigProvider using a simple string map.
type mapConfigProvider struct {
	values map[string]string
}

func (p *mapConfigProvider) GetString(key string) string        { return p.values[key] }
func (p *mapConfigProvider) GetInt(_ string) int                { return 0 }
func (p *mapConfigProvider) GetDuration(_ string) time.Duration { return 0 }

func clearSeedData(ctx context.Context, db *sql.DB, logger *slog.Logger) {
	if _, err := db.ExecContext(ctx, "DELETE FROM agent_logs WHERE id GLOB 'l0[0-9][0-9]'"); err != nil {
		logger.WarnContext(ctx, "clear seed logs", "error", err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM patch_history WHERE id GLOB 'h0[0-9][0-9]'"); err != nil {
		logger.WarnContext(ctx, "clear seed patch history", "error", err)
	}
}
