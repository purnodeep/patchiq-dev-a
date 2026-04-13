package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/skenzeriq/patchiq/internal/hub/api"
	hubv1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	hubauth "github.com/skenzeriq/patchiq/internal/hub/auth"
	"github.com/skenzeriq/patchiq/internal/hub/catalog"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/feeds"
	"github.com/skenzeriq/patchiq/internal/hub/store"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/hub/workers"
	"github.com/skenzeriq/patchiq/internal/shared/config"
	"github.com/skenzeriq/patchiq/internal/shared/idempotency"
	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
)

// Set by goreleaser via -ldflags.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("hub manager failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	startTime := time.Now()

	configPath := os.Getenv("PATCHIQ_HUB_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/hub.yaml"
	}
	cfg, err := config.LoadHub(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 1. Logger (before OTel so init errors are structured)
	var logLevel slog.Level
	var unknownLevel bool
	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "info", "":
		logLevel = slog.LevelInfo
	default:
		logLevel = slog.LevelInfo
		unknownLevel = true
	}
	logger := slog.New(piqotel.NewHandler(config.LogWriter(cfg.Log), &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	if unknownLevel {
		slog.Warn("unknown log level, defaulting to info", "configured_level", cfg.Log.Level)
	}

	// 2. OpenTelemetry
	otelShutdown, err := piqotel.Init(context.Background(), piqotel.Config{
		ServiceName:    "patchiq-hub",
		ServiceVersion: version,
		Environment:    cfg.Env,
		OTLPEndpoint:   cfg.OTel.Endpoint,
		Insecure:       cfg.OTel.Insecure,
	})
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}

	slog.Info("patchiq-hub", "version", version, "commit", commit)

	// 3. Context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 4. Database
	slog.Info("connecting to database")
	pool, err := store.NewPool(ctx, cfg.Database.URL, int32(cfg.Database.MaxConns), int32(cfg.Database.MinConns))
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	slog.Info("database connected", "max_conns", cfg.Database.MaxConns)

	// 5. Event bus (Watermill + PostgreSQL transport)
	wmLogger := watermill.NewSlogLogger(logger)
	wmDB := stdlib.OpenDBFromPool(pool)
	defer func() {
		if closeErr := wmDB.Close(); closeErr != nil {
			slog.Error("close watermill database connection", "error", closeErr)
		}
	}()

	pub, subFactory, err := events.NewPublisherAndSubscriberFactory(wmDB, wmLogger)
	if err != nil {
		return fmt.Errorf("create watermill pub/sub: %w", err)
	}

	wmRouter, err := events.NewRouter(wmLogger)
	if err != nil {
		return fmt.Errorf("create watermill router: %w", err)
	}

	eventBus := events.NewWatermillEventBus(pub, subFactory, wmRouter, wmLogger)

	auditSub := events.NewAuditSubscriber(pool, logger)
	if err := eventBus.Subscribe("*", auditSub.Handle); err != nil {
		return fmt.Errorf("subscribe audit handler: %w", err)
	}

	slog.Info("event bus initialized", "topic_count", len(events.AllTopics()))

	// 6. Feed aggregation (registry + pipeline + River)
	nvdAPIKey := os.Getenv("PATCHIQ_HUB_NVD_API_KEY")
	if nvdAPIKey == "" {
		slog.Warn("PATCHIQ_HUB_NVD_API_KEY not set; NVD feed will use unauthenticated rate limit (5 req/30s)")
	} else {
		slog.Info("NVD API key configured; using higher rate limit (50 req/30s)")
	}
	feedRegistry := map[string]feeds.Feed{
		"nvd":         feeds.NewNVDFeed(nil, nvdAPIKey),
		"cisa_kev":    feeds.NewCISAKEVFeed(nil),
		"msrc":        feeds.NewMSRCFeed(nil),
		"redhat_oval": feeds.NewRedHatFeed(nil),
		"ubuntu_usn":  feeds.NewUbuntuFeed(nil),
		"apple":       feeds.NewAppleFeed(nil),
	}

	queries := sqlcgen.New(pool)
	aptResolver := catalog.NewAPTPackageResolver()
	pipeline := catalog.NewPipeline(queries, eventBus, aptResolver)

	feedWorker := workers.NewFeedSyncWorker(feedRegistry, pipeline)
	// Binary fetch worker is always registered. When MinIO is not configured,
	// the downloader is nil and processFetch will mark pending rows as failed.
	binaryFetchWorker := workers.NewBinaryFetchWorker(queries, nil, eventBus)
	riverWorkers := workers.RegisterWorkers(feedWorker, binaryFetchWorker)

	// Build periodic jobs from DB-configured intervals.
	defaultIntervals := map[string]time.Duration{
		"nvd":         6 * time.Hour,
		"cisa_kev":    12 * time.Hour,
		"msrc":        12 * time.Hour,
		"redhat_oval": 12 * time.Hour,
		"ubuntu_usn":  12 * time.Hour,
		"apple":       12 * time.Hour,
	}

	feedSources, dbErr := queries.ListEnabledFeedSources(ctx)
	if dbErr != nil {
		slog.Warn("failed to load feed intervals from DB, using defaults", "error", dbErr)
	}

	dbIntervals := make(map[string]time.Duration, len(feedSources))
	for _, fs := range feedSources {
		if fs.SyncIntervalSeconds > 0 {
			dbIntervals[fs.Name] = time.Duration(fs.SyncIntervalSeconds) * time.Second
		}
	}

	var periodicJobs []*river.PeriodicJob
	for name := range feedRegistry {
		interval := dbIntervals[name]
		if interval == 0 {
			interval = defaultIntervals[name]
		}
		if interval == 0 {
			interval = 12 * time.Hour
			slog.Warn("no configured interval for feed, using 12h fallback", "feed", name)
		}
		feedName := name // capture for closure
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(interval),
			func() (river.JobArgs, *river.InsertOpts) {
				return workers.FeedSyncJobArgs{FeedName: feedName}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	periodicJobs = append(periodicJobs, river.NewPeriodicJob(
		river.PeriodicInterval(30*time.Minute),
		func() (river.JobArgs, *river.InsertOpts) {
			return workers.BinaryFetchJobArgs{}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: true},
	))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers:      riverWorkers,
		PeriodicJobs: periodicJobs,
	})
	if err != nil {
		return fmt.Errorf("create river client: %w", err)
	}

	if err := riverClient.Start(ctx); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}
	slog.Info("feed aggregation initialized", "feeds", len(feedRegistry), "periodic_jobs", len(periodicJobs))

	// 7. MinIO object store (optional — binary distribution)
	var binaryStore hubv1.BinaryStore
	minIOCfg := cfg.Hub.MinIO
	if minIOCfg.Endpoint != "" && minIOCfg.AccessKey != "" {
		ms, msErr := catalog.NewMinIOStore(minIOCfg.Endpoint, minIOCfg.AccessKey, minIOCfg.SecretKey, minIOCfg.UseSSL)
		if msErr != nil {
			slog.Warn("minio object store unavailable, binary distribution disabled",
				"endpoint", minIOCfg.Endpoint,
				"error", msErr,
			)
		} else {
			if ensureErr := ms.EnsureBucket(ctx, minIOCfg.Bucket); ensureErr != nil {
				slog.Warn("minio: could not ensure bucket, binary distribution may fail",
					"bucket", minIOCfg.Bucket,
					"error", ensureErr,
				)
			} else {
				slog.Info("minio object store connected",
					"endpoint", minIOCfg.Endpoint,
					"bucket", minIOCfg.Bucket,
				)
			}
			binaryStore = ms
			// Wire the binary downloader into the fetch worker now that MinIO is ready.
			fetcher := catalog.NewBinaryFetcher(ms, minIOCfg.Bucket, nil)
			binaryFetchWorker.SetDownloader(fetcher)
		}
	} else {
		slog.Warn("minio not configured, binary distribution disabled")
	}

	// 7.5. Authentication (Zitadel OIDC — slim: login + JWT only)
	var jwtMW func(http.Handler) http.Handler
	var loginHandler *hubauth.LoginHandler

	if cfg.IAM.Zitadel.ClientID != "" {
		scheme := "https"
		if !cfg.IAM.Zitadel.Secure {
			scheme = "http"
		}
		zitadelBaseURL := fmt.Sprintf("%s://%s", scheme, cfg.IAM.Zitadel.Domain)
		zitadelClient := hubauth.NewZitadelClient(zitadelBaseURL, cfg.IAM.Zitadel.ServiceAccountKey)

		sessionCfg := hubauth.SessionConfig{
			CookieName:      cfg.IAM.Session.CookieName,
			CookieDomain:    cfg.IAM.Session.CookieDomain,
			CookieSecure:    cfg.IAM.Session.CookieSecure,
			AccessTokenTTL:  cfg.IAM.Session.AccessTTL,
			RememberMeTTL:   cfg.IAM.Session.RememberMeTTL,
			DefaultTenantID: "00000000-0000-0000-0000-000000000001",
			PostLoginURL:    cfg.IAM.Session.PostLoginURL,
		}
		sessionCfg.InitSigningKey()

		loginHandler = hubauth.NewLoginHandler(zitadelClient, eventBus, sessionCfg)

		jwtMW = hubauth.NewJWTMiddleware(hubauth.JWTMiddlewareConfig{
			CookieName: cfg.IAM.Session.CookieName,
			SigningKey: sessionCfg.SigningKey,
			DevMode:    cfg.Env == "development",
		})

		slog.Info("hub auth initialized", "zitadel_domain", cfg.IAM.Zitadel.Domain)
	} else {
		slog.Warn("hub auth not configured — using header-based auth stubs (not suitable for production)")
	}

	// 8. HTTP server
	syncAPIKey := os.Getenv("PATCHIQ_HUB_SYNC_API_KEY")
	if syncAPIKey == "" {
		slog.Warn("PATCHIQ_HUB_SYNC_API_KEY not set, sync endpoint will reject all requests")
	}

	// TODO(PIQ-14): replace with ValkeyStore when Valkey client is wired in hub startup.
	idempotencyStore := idempotency.NewMemoryStore()
	slog.Warn("using in-memory idempotency store, cached responses will not survive restarts")

	healthChecks := map[string]hubv1.CheckFunc{
		"watermill": func(_ context.Context) error {
			if !wmRouter.IsRunning() {
				return fmt.Errorf("watermill router is not running")
			}
			return nil
		},
	}

	router := api.NewRouter(pool, eventBus, syncAPIKey, startTime, version, idempotencyStore, cfg.Hub.CORSOrigins, riverClient, binaryStore, minIOCfg.Bucket, jwtMW, loginHandler, cfg.Hub.DefaultTenantID, healthChecks)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Hub.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.Hub.HTTP.ReadTimeout,
		WriteTimeout: cfg.Hub.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Hub.HTTP.IdleTimeout,
	}

	// Start servers
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("starting http server", "port", cfg.Hub.HTTP.Port)
		if listenErr := httpServer.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", listenErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if runErr := wmRouter.Run(ctx); runErr != nil {
			errCh <- fmt.Errorf("event bus router: %w", runErr)
		}
	}()

	slog.Info("hub manager started",
		"http_port", cfg.Hub.HTTP.Port,
		"env", cfg.Env,
	)

	// Wait for shutdown signal or error
	var serverErr error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case serverErr = <-errCh:
		slog.Error("server error, initiating shutdown", "error", serverErr)
	}

	// Ordered shutdown: HTTP -> Watermill -> OTel; DB pool closed via defer
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Hub.ShutdownTimeout)
	defer cancel()

	var shutdownErrs []error

	slog.Info("shutting down http server")
	if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("http shutdown error", "error", shutdownErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("http shutdown: %w", shutdownErr))
	}

	slog.Info("shutting down river client")
	if stopErr := riverClient.Stop(shutdownCtx); stopErr != nil {
		slog.ErrorContext(shutdownCtx, "river client shutdown error", "error", stopErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("river client shutdown: %w", stopErr))
	}

	slog.Info("shutting down event bus")
	if closeErr := eventBus.Close(); closeErr != nil {
		slog.Error("event bus shutdown error", "error", closeErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("event bus shutdown: %w", closeErr))
	}

	slog.Info("shutting down otel")
	if otelErr := otelShutdown(shutdownCtx); otelErr != nil {
		slog.Error("otel shutdown error", "error", otelErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("otel shutdown: %w", otelErr))
	}

	wg.Wait()
	close(errCh)
	for drainErr := range errCh {
		slog.Error("additional server error during shutdown", "error", drainErr)
	}

	slog.Info("hub manager stopped")
	if serverErr != nil {
		return serverErr
	}
	return errors.Join(shutdownErrs...)
}
