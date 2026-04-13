package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/valkey-io/valkey-go"
	"google.golang.org/grpc/reflection"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	patchiqv1 "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/api"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/server/compliance"
	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/discovery"
	"github.com/skenzeriq/patchiq/internal/server/events"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/reports"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/targeting"
	"github.com/skenzeriq/patchiq/internal/server/workers"
	"github.com/skenzeriq/patchiq/internal/server/workflow"
	"github.com/skenzeriq/patchiq/internal/shared/config"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
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
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	startTime := time.Now()

	// 1. Config
	configPath := os.Getenv("PATCHIQ_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/server.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Logger (before OTel so init errors are structured)
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

	// 3. OpenTelemetry
	otelShutdown, err := piqotel.Init(context.Background(), piqotel.Config{
		ServiceName:    "patchiq-server",
		ServiceVersion: version,
		Environment:    cfg.Env,
		OTLPEndpoint:   cfg.OTel.Endpoint,
		Insecure:       cfg.OTel.Insecure,
	})
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}

	slog.Info("patchiq-server", "version", version, "commit", commit)

	// 4. Context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 5. Database
	slog.Info("connecting to database")
	pool, err := store.NewPool(ctx, cfg.Database.URL, int32(cfg.Database.MaxConns), int32(cfg.Database.MinConns))
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	slog.Info("database connected", "max_conns", cfg.Database.MaxConns)

	st := store.NewStore(pool)

	// Look up default tenant for background jobs and JWT fallback.
	var defaultTenantID string
	row := pool.QueryRow(context.Background(), "SELECT id FROM tenants ORDER BY id LIMIT 1")
	if err := row.Scan(&defaultTenantID); err != nil {
		slog.Warn("could not resolve default tenant", "error", err)
	}

	// 6. Event bus (Watermill + PostgreSQL transport)
	wmLogger := watermill.NewSlogLogger(logger)
	wmDB := stdlib.OpenDBFromPool(pool)
	defer wmDB.Close()

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

	alertSub := events.NewAlertSubscriber(pool, logger)
	alertSub.StartCacheRefresh(ctx, 30*time.Second)
	if err := eventBus.Subscribe("*", alertSub.Handle); err != nil {
		return fmt.Errorf("subscribe alert handler: %w", err)
	}
	slog.Info("alert subscriber initialized")

	slog.Info("event bus initialized", "topic_count", len(events.AllTopics()))

	// 7. Discovery engine
	discoveryCfg := config.DefaultDiscoveryConfig()
	httpTimeout := 30
	if discoveryCfg.HTTPTimeout != nil {
		httpTimeout = *discoveryCfg.HTTPTimeout
	}
	maxRetries := 3
	if discoveryCfg.MaxRetries != nil {
		maxRetries = *discoveryCfg.MaxRetries
	}
	fetcher := discovery.NewFetcher(time.Duration(httpTimeout)*time.Second, maxRetries)
	storeAdapter := discovery.NewStoreAdapter(pool)
	eventAdapter := discovery.NewEventAdapter(eventBus)
	discoverySvc := discovery.NewService(storeAdapter, eventAdapter, fetcher)
	discoveryWorker := discovery.NewDiscoveryWorker(discoverySvc, discoveryCfg)

	// 7b. CVE engine
	cveCfg := config.DefaultCVEConfig()
	nvdAPIKey := os.Getenv("PATCHIQ_NVD_API_KEY")
	cveHTTPTimeout := 60
	if cveCfg.HTTPTimeout != nil {
		cveHTTPTimeout = *cveCfg.HTTPTimeout
	}
	// Use HubCVEClient when Hub is configured, else fall back to direct NVD.
	// Hub URL and API key are read from env vars (same as hub sync bootstrap below).
	var cveFetcher cve.CVEFetcher
	hubCVEURL := os.Getenv("PATCHIQ_HUB_URL")
	hubCVEAPIKey := os.Getenv("PATCHIQ_HUB_API_KEY")
	if hubCVEURL != "" && hubCVEAPIKey != "" {
		cveFetcher = cve.NewHubCVEClient(hubCVEURL, hubCVEAPIKey)
		slog.Info("cve engine: using Hub as CVE source", "hub_url", hubCVEURL)
	} else {
		cveFetcher = cve.NewNVDClient("", nvdAPIKey, time.Duration(cveHTTPTimeout)*time.Second)
		slog.Info("cve engine: using direct NVD as CVE source")
	}
	cveStore := cve.NewStoreAdapter(pool)
	cveEvents := cve.NewEventAdapter(eventBus)
	cveCorrelator := cve.NewCorrelator(cveStore, cveStore)
	cveSyncSvc := cve.NewNVDSyncService(cveFetcher, cveStore, cveEvents, cveCorrelator)
	cveMatcher := cve.NewMatcher(cveStore, cveStore, cveStore).WithOsFamilyLookup(cveStore)

	nvdSyncWorker := cve.NewNVDSyncWorker(cveSyncSvc)
	endpointMatchWorker := cve.NewEndpointMatchWorker(cveMatcher).WithOsFamilyGetter(cveStore)

	// Post-sync matching is wired after River client creation (see below).

	// 7c. Deployment engine
	deployQ := sqlcgen.New(pool)
	// Tag selector resolver is shared across both the deployment and
	// policy evaluators — a single Resolver instance, one pgxpool, one
	// set of RLS-scoped transactions per call.
	targetingResolver := targeting.NewResolver(pool)
	deployEvaluator := deployment.NewEvaluator(targetingResolver)
	deploySM := deployment.NewStateMachine()
	deployExecutor := deployment.NewExecutor(deployQ, deploySM, eventBus)
	executorWorker := deployment.NewExecutorWorker(deployExecutor)
	timeoutTxFactory := newTimeoutTxFactory(pool)
	deployTimeoutChecker := deployment.NewTimeoutChecker(deployQ, deploySM, eventBus, deployment.WithTimeoutTxFactory(timeoutTxFactory))
	timeoutWorker := deployment.NewTimeoutWorker(deployTimeoutChecker)
	scanScheduler := deployment.NewScanScheduler(deployQ, eventBus)
	scanWorker := deployment.NewScanWorker(scanScheduler, deployQ)

	// 7e. Wave dispatcher
	waveDispatcherTxFactory := newWaveDispatcherTxFactory(pool)
	waveDispatcher := deployment.NewWaveDispatcher(deployQ, deploySM, eventBus, 30*time.Minute, deployment.WithWaveDispatcherTxFactory(waveDispatcherTxFactory))
	waveDispatcherWorker := deployment.NewWaveDispatcherWorker(waveDispatcher)

	// 7f. Schedule checker
	scheduleChecker := deployment.NewScheduleChecker(deployQ, eventBus)
	scheduleCheckerWorker := deployment.NewScheduleCheckerWorker(scheduleChecker)

	// 7g. Result handler (processes command results from agents)
	resultTxFactory := newResultTxFactory(pool)
	resultHandler := deployment.NewResultHandler(deployQ, deploySM, eventBus, deployment.WithResultTxFactory(resultTxFactory))
	if err := eventBus.Subscribe(events.CommandResultReceived, func(ctx context.Context, evt domain.DomainEvent) error {
		var payload events.CommandResultPayload
		switch p := evt.Payload.(type) {
		case events.CommandResultPayload:
			payload = p
		default:
			// After JSON round-trip through Watermill, Payload arrives as map[string]interface{}.
			// Re-marshal and unmarshal to recover the typed struct.
			raw, jErr := json.Marshal(evt.Payload)
			if jErr != nil {
				slog.ErrorContext(ctx, "result handler: marshal payload", "event_id", evt.ID, "error", jErr)
				return fmt.Errorf("result handler: marshal payload: %w", jErr)
			}
			if jErr = json.Unmarshal(raw, &payload); jErr != nil {
				slog.ErrorContext(ctx, "result handler: unmarshal payload", "event_id", evt.ID, "error", jErr)
				return fmt.Errorf("result handler: unmarshal payload: %w", jErr)
			}
		}

		commandIDStr := payload.CommandID
		if commandIDStr == "" {
			return fmt.Errorf("result handler: missing command_id in event payload")
		}

		var commandID pgtype.UUID
		if parseErr := commandID.Scan(commandIDStr); parseErr != nil {
			slog.ErrorContext(ctx, "result handler: invalid command_id", "command_id", commandIDStr, "error", parseErr)
			return fmt.Errorf("result handler: invalid command_id: %w", parseErr)
		}
		var tenantID pgtype.UUID
		if parseErr := tenantID.Scan(evt.TenantID); parseErr != nil {
			slog.ErrorContext(ctx, "result handler: invalid tenant_id", "tenant_id", evt.TenantID, "error", parseErr)
			return fmt.Errorf("result handler: invalid tenant_id: %w", parseErr)
		}

		return resultHandler.HandleResult(ctx, commandID, tenantID, payload.Succeeded, payload.Output, payload.Stderr, payload.ErrorMessage, payload.ExitCode)
	}); err != nil {
		return fmt.Errorf("subscribe result handler: %w", err)
	}
	slog.Info("result handler subscribed", "topic", events.CommandResultReceived)

	// 7e. Notification engine
	notifyCryptoKeyHex := os.Getenv("PATCHIQ_NOTIFICATION_KEY")
	var notifyCryptoKey []byte
	if len(notifyCryptoKeyHex) == 64 {
		var decodeErr error
		notifyCryptoKey, decodeErr = hex.DecodeString(notifyCryptoKeyHex)
		if decodeErr != nil || len(notifyCryptoKey) != crypto.KeySize {
			if cfg.Env == "production" {
				return fmt.Errorf("PATCHIQ_NOTIFICATION_KEY is invalid (must be 32-byte hex): production requires a valid key")
			}
			slog.Warn("PATCHIQ_NOTIFICATION_KEY invalid, generating ephemeral key")
			notifyCryptoKey = crypto.GenerateKey()
		}
	} else {
		if cfg.Env == "production" {
			return fmt.Errorf("PATCHIQ_NOTIFICATION_KEY not set or wrong length (expected 64 hex chars): production requires a valid key")
		}
		slog.Warn("PATCHIQ_NOTIFICATION_KEY not set or wrong length, generating ephemeral key (not suitable for production)")
		notifyCryptoKey = crypto.GenerateKey()
	}

	notifySender := &notify.ShoutrrrSender{}
	notifyRecorder := notify.NewDBHistoryRecorder(pool)
	notifySendWorker := notify.NewSendWorker(notifySender, notifyRecorder, eventBus)

	// 7f. Audit retention
	retentionQ := sqlcgen.New(pool)
	partitionDropper := workers.NewPgPartitionDropper(pool)
	retentionPurger := workers.NewAuditRetentionPurger(retentionQ, partitionDropper)
	retentionWorker := workers.NewAuditRetentionWorker(retentionPurger)

	// 7h. Compliance evaluation — delegates to compliance.Service with RLS-safe transactions.
	complianceCustomFWLoader := compliance.NewDBCustomFrameworkLoader(sqlcgen.New(pool))
	complianceWorkerSvc := compliance.NewService().WithCustomFrameworks(complianceCustomFWLoader).WithControlQuerier(sqlcgen.New(pool))
	complianceTenantLister := sqlcgen.New(pool)
	complianceEvaluator := workers.NewComplianceEvaluator(complianceTenantLister, complianceWorkerSvc, st, eventBus)
	complianceEvalWorker := workers.NewComplianceEvalWorker(complianceEvaluator)

	// 7i. Catalog sync worker
	catalogSyncStore := sqlcgen.New(pool)
	catalogSyncWorker := workers.NewCatalogSyncWorker(catalogSyncStore, pool, eventBus)
	catalogSyncWorker.WithCVESync(cveSyncSvc)

	// 7j. Workflow execution worker
	workflowHandlers := buildWorkflowHandlers(pool)
	workflowExecWorker := workflow.NewWorkflowExecuteWorker(pool, eventBus, workflowHandlers)

	// 7k. User sync and policy scheduler workers — nil until their interface
	// dependencies are implemented. addWorkerIfNotNil in RegisterWorkers skips
	// nil workers with a warning log.
	var userSyncWorker *workers.UserSyncWorker              // TODO(PIQ-321): instantiate when Zitadel user lister is implemented
	var policySchedulerWorker *policy.PolicySchedulerWorker // TODO(PIQ-321): instantiate when policy data source is implemented

	// 8. River (background jobs)
	var periodicJobs []*river.PeriodicJob

	// Auto-bootstrap hub_sync_state for the default tenant when Hub URL + API key
	// are configured via environment. This allows catalog sync to start immediately
	// without requiring a manual PUT /api/v1/sync/config call.
	hubURL := os.Getenv("PATCHIQ_HUB_URL")
	hubAPIKey := os.Getenv("PATCHIQ_HUB_API_KEY")
	if hubURL != "" && hubAPIKey != "" && defaultTenantID != "" {
		var bootstrapTenantID pgtype.UUID
		if scanErr := bootstrapTenantID.Scan(defaultTenantID); scanErr == nil {
			// Use a background context so startup errors don't block the server.
			bootstrapCtx := context.Background()
			// RLS requires app.current_tenant_id — wrap in a transaction.
			bootstrapTx, txErr := pool.Begin(bootstrapCtx)
			if txErr != nil {
				slog.Warn("hub sync: could not begin bootstrap transaction", "error", txErr)
			} else {
				_, _ = bootstrapTx.Exec(bootstrapCtx,
					fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", defaultTenantID))
				bootstrapQ := sqlcgen.New(bootstrapTx)
				_, upsertErr := bootstrapQ.UpsertHubSyncState(bootstrapCtx, sqlcgen.UpsertHubSyncStateParams{
					TenantID:     bootstrapTenantID,
					HubUrl:       hubURL,
					ApiKey:       hubAPIKey,
					SyncInterval: 21600, // 6 hours
				})
				if upsertErr != nil {
					_ = bootstrapTx.Rollback(bootstrapCtx)
					slog.Warn("hub sync: could not bootstrap hub_sync_state", "error", upsertErr)
				} else if commitErr := bootstrapTx.Commit(bootstrapCtx); commitErr != nil {
					slog.Warn("hub sync: could not commit hub_sync_state bootstrap", "error", commitErr)
				} else {
					slog.Info("hub sync: bootstrapped hub_sync_state",
						"hub_url", hubURL,
						"tenant_id", defaultTenantID,
					)
				}
			}
		}
	}

	// Discovery periodic job
	discoveryInterval := 60 // default 60 minutes
	if discoveryCfg.SyncIntervalMins != nil {
		discoveryInterval = *discoveryCfg.SyncIntervalMins
	}
	if discoveryCfg.Schedule != nil {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(discoveryInterval)*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return discovery.DiscoveryJobArgs{TenantID: defaultTenantID}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
		slog.Info("discovery periodic job registered",
			"interval_mins", discoveryInterval,
		)
	}

	// CVE NVD sync periodic job
	cveInterval := 1440 // default 24 hours
	if cveCfg.SyncIntervalMins != nil {
		cveInterval = *cveCfg.SyncIntervalMins
	}
	if cveCfg.Schedule != nil {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(cveInterval)*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return cve.NVDSyncJobArgs{TenantID: defaultTenantID}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
		slog.Info("cve nvd sync periodic job registered",
			"interval_mins", cveInterval,
		)
	}

	// Catalog sync periodic job: run every 6 hours when Hub is configured.
	if hubURL != "" && hubAPIKey != "" {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(6*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return workers.CatalogSyncJobArgs{}, nil
			},
			// RunOnStart triggers a sync shortly after server startup so
			// catalog entries are available without waiting 6 hours.
			&river.PeriodicJobOpts{RunOnStart: true},
		))
		slog.Info("catalog sync periodic job registered", "interval_hours", 6)
	}

	// User sync periodic job: run every 15 minutes when worker is available.
	if userSyncWorker != nil {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(15*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return workers.UserSyncJobArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
		slog.Info("user sync periodic job registered", "interval_mins", 15)
	}

	// Policy scheduler periodic job: run every 5 minutes when worker is available.
	if policySchedulerWorker != nil {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(5*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return policy.PolicySchedulerJobArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
		slog.Info("policy scheduler periodic job registered", "interval_mins", 5)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			"critical":   {MaxWorkers: max(cfg.River.MaxWorkers*40/100, 1)},
			"default":    {MaxWorkers: max(cfg.River.MaxWorkers*40/100, 1)},
			"background": {MaxWorkers: max(cfg.River.MaxWorkers*20/100, 1)},
		},
		Workers: workers.RegisterWorkers(discoveryWorker, nvdSyncWorker, endpointMatchWorker, executorWorker, timeoutWorker, scanWorker, notifySendWorker, waveDispatcherWorker, scheduleCheckerWorker, retentionWorker, complianceEvalWorker, userSyncWorker, catalogSyncWorker, workflowExecWorker, policySchedulerWorker),
		PeriodicJobs: append(periodicJobs,
			river.NewPeriodicJob(
				river.PeriodicInterval(5*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return deployment.TimeoutJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return deployment.ScanJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(60*time.Second),
				func() (river.JobArgs, *river.InsertOpts) {
					return deployment.WaveDispatcherJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(1*time.Minute),
				func() (river.JobArgs, *river.InsertOpts) {
					return deployment.ScheduleCheckerJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return workers.AuditRetentionJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(6*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return workers.ComplianceEvalJobArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
		),
	})
	if err != nil {
		return fmt.Errorf("create river client: %w", err)
	}
	slog.Info("river worker pool initialized",
		"critical_workers", max(cfg.River.MaxWorkers*40/100, 1),
		"default_workers", max(cfg.River.MaxWorkers*40/100, 1),
		"background_workers", max(cfg.River.MaxWorkers*20/100, 1),
	)

	// Wire post-sync matching: after NVD sync completes, enqueue match jobs for all endpoints.
	nvdSyncWorker.WithPostSyncMatching(cveStore, riverClient)

	// Discovery enqueuer + handler
	discoveryEnqueuer := discovery.NewEnqueuer(riverClient)
	discoveryHandler := v1.NewDiscoveryHandler(discoveryEnqueuer)

	// Notification event handler (Watermill subscriptions)
	notifyResolver := notify.NewDBPreferenceResolver(pool, notifyCryptoKey)
	notifyEnqueuer := notify.NewRiverEnqueuer(riverClient)
	notifyHandler := events.NewNotificationHandler(notifyResolver, notifyEnqueuer)

	notifyTopics := []string{
		events.DeploymentStarted,
		events.DeploymentCompleted,
		events.DeploymentFailed,
		events.DeploymentRollbackTriggered,
		events.ComplianceThresholdBreach,
		events.ComplianceEvaluationCompleted,
		events.AgentDisconnected,
		events.CVEDiscovered,
		events.CVERemediationAvailable,
		events.CatalogSyncFailed,
		events.LicenseExpiring,
	}
	for _, triggerTopic := range notifyTopics {
		if err := eventBus.Subscribe(triggerTopic, notifyHandler.Handle); err != nil {
			return fmt.Errorf("subscribe notification handler to %s: %w", triggerTopic, err)
		}
	}
	slog.Info("notification handler subscribed", "trigger_count", len(notifyTopics))

	// CVE endpoint match trigger (enqueue matching job when inventory arrives)
	if err := eventBus.Subscribe(events.InventoryScanCompleted, func(ctx context.Context, evt domain.DomainEvent) error {
		payload, ok := evt.Payload.(map[string]any)
		if !ok {
			slog.ErrorContext(ctx, "cve match trigger: unexpected event payload type", "event_id", evt.ID)
			return fmt.Errorf("cve match trigger: unexpected payload type %T", evt.Payload)
		}
		endpointID, _ := payload["endpoint_id"].(string)
		if endpointID == "" {
			slog.ErrorContext(ctx, "cve match trigger: missing endpoint_id in payload", "event_id", evt.ID)
			return fmt.Errorf("cve match trigger: missing endpoint_id in event payload")
		}
		tenantID := evt.TenantID
		if tenantID == "" {
			slog.ErrorContext(ctx, "cve match trigger: missing tenant_id", "event_id", evt.ID)
			return fmt.Errorf("cve match trigger: missing tenant_id in event")
		}

		_, insertErr := riverClient.Insert(ctx, cve.EndpointMatchJobArgs{
			TenantID:   tenantID,
			EndpointID: endpointID,
		}, nil)
		if insertErr != nil {
			slog.ErrorContext(ctx, "cve match trigger: failed to enqueue job",
				"endpoint_id", endpointID, "tenant_id", tenantID, "error", insertErr)
			return fmt.Errorf("cve match trigger: enqueue endpoint match job: %w", insertErr)
		}
		slog.InfoContext(ctx, "cve match trigger: enqueued endpoint match job",
			"endpoint_id", endpointID, "tenant_id", tenantID)
		return nil
	}); err != nil {
		return fmt.Errorf("subscribe cve match trigger: %w", err)
	}
	slog.Info("cve match trigger subscribed", "topic", events.InventoryScanCompleted)

	// 9. gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPC.Port))
	if err != nil {
		return fmt.Errorf("listen grpc port %d: %w", cfg.Server.GRPC.Port, err)
	}
	grpcServer := servergrpc.NewGRPCServer(servergrpc.ServerConfig{})
	agentSvc := servergrpc.NewAgentServiceServer(st, eventBus, logger)
	agentSvc.SetCVEJobInserter(riverClient)
	patchiqv1.RegisterAgentServiceServer(grpcServer, agentSvc)
	if cfg.Server.GRPC.Reflection {
		reflection.Register(grpcServer)
		slog.Info("grpc reflection enabled")
	}

	// 10. HTTP server
	var idempotencyStore idempotency.Store
	var valkeyClient valkey.Client
	valkeyURL := cfg.Valkey.URL
	if valkeyURL == "" {
		valkeyURL = os.Getenv("PATCHIQ_VALKEY_URL")
	}
	if valkeyURL != "" {
		vc, valkeyErr := valkey.NewClient(valkey.ClientOption{
			InitAddress: []string{valkeyURL},
		})
		if valkeyErr != nil {
			slog.Warn("valkey connection failed, falling back to in-memory idempotency store",
				"url", valkeyURL,
				"error", valkeyErr,
			)
			idempotencyStore = idempotency.NewMemoryStore()
		} else {
			valkeyClient = vc
			idempotencyStore = idempotency.NewValkeyStore(valkeyClient)
			slog.Info("idempotency store using valkey", "url", valkeyURL)
		}
	} else {
		slog.Warn("valkey URL not configured, using in-memory idempotency store (not suitable for production)")
		idempotencyStore = idempotency.NewMemoryStore()
	}
	if hubURL == "" {
		slog.Warn("PATCHIQ_HUB_URL not set, hub sync will be unavailable")
	}
	if hubAPIKey == "" {
		slog.Warn("PATCHIQ_HUB_API_KEY not set, hub sync will be unavailable")
	}
	deploymentHandler := v1.NewDeploymentHandler(deployQ, pool, riverClient, eventBus, deployEvaluator, deploySM)
	scheduleHandler := v1.NewScheduleHandler(deployQ, eventBus)
	notificationAPIHandler := v1.NewNotificationHandler(deployQ, pool, notifyCryptoKey, notifySender, eventBus, notifyEnqueuer)
	notifByTypeHandler := v1.NewNotificationByTypeHandler(deployQ, notifyCryptoKey, eventBus, notifySender)
	customFWLoader := compliance.NewDBCustomFrameworkLoader(sqlcgen.New(pool))
	complianceSvc := compliance.NewService().WithCustomFrameworks(customFWLoader).WithControlQuerier(sqlcgen.New(pool))
	complianceHandler := v1.NewComplianceHandler(deployQ, complianceSvc, eventBus, st)
	hubSyncAPIHandler := v1.NewHubSyncAPIHandler(sqlcgen.New(pool), riverClient, eventBus)

	// IAM settings and role mapping handlers are always available
	// (they read/write DB settings, independent of Zitadel connectivity).
	iamQ := sqlcgen.New(pool)
	iamSettingsHandler := v1.NewIAMSettingsHandler(iamQ, notifyCryptoKey, eventBus)
	roleMappingHandler := v1.NewRoleMappingHandler(iamQ, pool, eventBus, nil)
	generalSettingsHandler := v1.NewGeneralSettingsHandler(iamQ, eventBus)

	// Initialize Zitadel OIDC auth if configured.
	var jwtMW func(http.Handler) http.Handler
	var ssoHandler *auth.SSOHandler
	var loginHandler *auth.LoginHandler
	var inviteHandler *auth.InviteHandler

	if cfg.IAM.Zitadel.ClientID != "" {
		scheme := "https"
		if !cfg.IAM.Zitadel.Secure {
			scheme = "http"
		}
		issuer := fmt.Sprintf("%s://%s", scheme, cfg.IAM.Zitadel.Domain)

		jwtCfg := auth.JWTConfig{
			Issuer:          issuer,
			JWKSURL:         issuer + "/oauth/v2/keys",
			CookieName:      cfg.IAM.Session.CookieName,
			DefaultTenantID: defaultTenantID,
			DevMode:         cfg.Env == "development",
		}
		jwtMW = auth.NewJWTMiddleware(jwtCfg)

		ssoCfg := auth.SSOConfig{
			ZitadelDomain: cfg.IAM.Zitadel.Domain,
			ZitadelSecure: cfg.IAM.Zitadel.Secure,
			ClientID:      cfg.IAM.Zitadel.ClientID,
			ClientSecret:  cfg.IAM.Zitadel.ClientSecret,
			RedirectURI:   cfg.IAM.Zitadel.RedirectURI,
			CookieName:    cfg.IAM.Session.CookieName,
			CookieDomain:  cfg.IAM.Session.CookieDomain,
			CookieSecure:  cfg.IAM.Session.CookieSecure,
			AccessTTL:     cfg.IAM.Session.AccessTTL,
			RefreshTTL:    cfg.IAM.Session.RefreshTTL,
			PostLoginURL:  cfg.IAM.Session.PostLoginURL,
		}
		ssoHandler = auth.NewSSOHandler(ssoCfg, nil)

		// Zitadel API client for direct login and invite flows.
		zitadelBaseURL := fmt.Sprintf("%s://%s", scheme, cfg.IAM.Zitadel.Domain)
		pat := cfg.IAM.Zitadel.ServiceAccountKey
		zitadelClient := auth.NewZitadelClient(zitadelBaseURL, pat)
		zitadelClient.SetOIDCCredentials(cfg.IAM.Zitadel.ClientID, cfg.IAM.Zitadel.ClientSecret)

		sessionCfg := auth.SessionConfig{
			CookieName:      cfg.IAM.Session.CookieName,
			CookieDomain:    cfg.IAM.Session.CookieDomain,
			CookieSecure:    cfg.IAM.Session.CookieSecure,
			AccessTokenTTL:  cfg.IAM.Session.AccessTTL,
			RememberMeTTL:   cfg.IAM.Session.RememberMeTTL,
			DefaultTenantID: defaultTenantID,
		}

		// InitSigningKey must be called before NewLoginHandler so the same key
		// is embedded in loginHandler.cfg and exposed via LocalSigningKey.
		// Calling it after would cause loginHandler to lazily generate a
		// different key at request time, making HMAC validation fail with 401.
		sessionCfg.InitSigningKey()
		auth.SetLocalSigningKey(sessionCfg.SigningKey)

		loginHandler = auth.NewLoginHandler(zitadelClient, eventBus, sessionCfg)

		inviteQ := sqlcgen.New(pool)
		inviteHandler = auth.NewInviteHandler(inviteQ, zitadelClient, eventBus, sessionCfg, cfg.IAM.Session.PostLoginURL)

		slog.Info("iam initialized", "zitadel_domain", cfg.IAM.Zitadel.Domain)
	} else {
		slog.Warn("iam not configured — using header-based auth stubs (not suitable for production)")
	}

	// Reports service (optional — requires MinIO)
	var reportH *v1.ReportHandler
	minioCfg := cfg.Server.MinIO
	if minioCfg.Endpoint != "" && minioCfg.AccessKey != "" {
		mc, mcErr := minio.New(minioCfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
			Secure: minioCfg.UseSSL,
		})
		if mcErr != nil {
			slog.Warn("minio unavailable, reports disabled", "endpoint", minioCfg.Endpoint, "error", mcErr)
		} else {
			// Ensure bucket exists.
			bucket := minioCfg.Bucket
			if bucket == "" {
				bucket = "patchiq-reports"
			}
			if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				// Ignore "bucket already owned" error.
				if exists, _ := mc.BucketExists(ctx, bucket); !exists {
					slog.Warn("minio: could not ensure bucket", "bucket", bucket, "error", err)
				}
			}

			reportQ := sqlcgen.New(pool)
			storeAdapter := reports.NewStoreAdapter(reportQ)
			reportSvc := reports.NewService(storeAdapter, mc, bucket)

			reportSvc.RegisterAssembler(reports.ReportEndpoints, func() reports.Assembler {
				return reports.NewEndpointsAssembler(reportQ)
			})
			reportSvc.RegisterAssembler(reports.ReportPatches, func() reports.Assembler {
				return reports.NewPatchesAssembler(reportQ)
			})
			reportSvc.RegisterAssembler(reports.ReportCVEs, func() reports.Assembler {
				return reports.NewCVEsAssembler(reportQ)
			})
			reportSvc.RegisterAssembler(reports.ReportDeployments, func() reports.Assembler {
				return reports.NewDeploymentsAssembler(reportQ)
			})
			reportSvc.RegisterAssembler(reports.ReportCompliance, func() reports.Assembler {
				return reports.NewComplianceAssembler(reportQ)
			})
			reportSvc.RegisterAssembler(reports.ReportExecutive, func() reports.Assembler {
				return reports.NewExecutiveAssembler(reportQ)
			})

			reportSvc.RegisterRenderer(reports.FormatPDF, &reports.PDFRenderer{})
			reportSvc.RegisterRenderer(reports.FormatCSV, &reports.CSVRenderer{})
			reportSvc.RegisterRenderer(reports.FormatXLSX, &reports.XLSXRenderer{})

			reportH = v1.NewReportHandler(reportSvc)
			slog.Info("reports service initialized", "bucket", bucket, "endpoint", minioCfg.Endpoint)
		}
	} else {
		slog.Warn("minio not configured, reports disabled")
	}
	_ = reportH // TODO(PIQ-334): wire report handler into router once reports feature is complete

	// Build health checks for readiness endpoint.
	healthChecks := make(map[string]v1.CheckFunc)
	if valkeyClient != nil {
		healthChecks["valkey"] = func(ctx context.Context) error {
			return valkeyClient.Do(ctx, valkeyClient.B().Ping().Build()).Error()
		}
	}
	healthChecks["watermill"] = func(_ context.Context) error {
		if !wmRouter.IsRunning() {
			return fmt.Errorf("watermill router is not running")
		}
		return nil
	}

	agentBinariesDir := os.Getenv("PATCHIQ_SERVER_AGENT_BINARIES_DIR")
	if agentBinariesDir == "" {
		agentBinariesDir = "dist/agents"
	}
	agentGRPCHost := os.Getenv("PATCHIQ_SERVER_AGENT_GRPC_HOST")
	if agentGRPCHost == "" {
		if h, err := os.Hostname(); err == nil {
			agentGRPCHost = h
		} else {
			agentGRPCHost = "localhost"
		}
	}
	agentBinariesHandler := v1.NewAgentBinariesHandler(agentBinariesDir, fmt.Sprintf("%s:%d", agentGRPCHost, cfg.Server.GRPC.Port))

	router := api.NewRouter(st, eventBus, hubURL, hubAPIKey, startTime, idempotencyStore, version, discoveryHandler, deploymentHandler, scheduleHandler, scanScheduler, nil, cfg.Server.CORSOrigins, notificationAPIHandler, complianceHandler, jwtMW, ssoHandler, iamSettingsHandler, roleMappingHandler, hubSyncAPIHandler, riverClient, notifByTypeHandler, generalSettingsHandler, loginHandler, inviteHandler, alertSub, agentBinariesHandler, reportH, healthChecks)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
		IdleTimeout:  cfg.Server.HTTP.IdleTimeout,
	}

	// Start all servers
	errCh := make(chan error, 4)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("starting grpc server", "port", cfg.Server.GRPC.Port)
		if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
			errCh <- fmt.Errorf("grpc server: %w", serveErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("starting http server", "port", cfg.Server.HTTP.Port)
		if listenErr := httpServer.ListenAndServe(); listenErr != nil && listenErr != http.ErrServerClosed {
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		if startErr := riverClient.Start(ctx); startErr != nil {
			errCh <- fmt.Errorf("river client: %w", startErr)
		}
	}()

	slog.Info("patch manager started",
		"http_port", cfg.Server.HTTP.Port,
		"grpc_port", cfg.Server.GRPC.Port,
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

	// Ordered shutdown (HTTP → gRPC → River → Watermill); DB pool closed via defer
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	var shutdownErrs []error

	slog.Info("shutting down http server")
	if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("http shutdown error", "error", shutdownErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("http shutdown: %w", shutdownErr))
	}

	slog.Info("shutting down grpc server")
	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()
	select {
	case <-grpcDone:
	case <-shutdownCtx.Done():
		slog.Warn("grpc graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}

	slog.Info("shutting down river")
	if stopErr := riverClient.Stop(shutdownCtx); stopErr != nil {
		slog.Error("river shutdown error", "error", stopErr)
		shutdownErrs = append(shutdownErrs, fmt.Errorf("river shutdown: %w", stopErr))
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

	slog.Info("patch manager stopped")
	if serverErr != nil {
		return serverErr
	}
	return errors.Join(shutdownErrs...)
}

// beginTenantTx opens a tenant-scoped transaction, sets the RLS tenant context,
// and returns the querier, commit, and rollback functions.
func beginTenantTx[Q any](ctx context.Context, pool *pgxpool.Pool, tenantID string, newQuerier func(*sqlcgen.Queries) Q, errPrefix string) (Q, func() error, func() error, error) {
	var zero Q
	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, nil, nil, fmt.Errorf("%s: begin: %w", errPrefix, err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.WarnContext(ctx, errPrefix+": rollback after set_config failure", "error", rbErr)
		}
		return zero, nil, nil, fmt.Errorf("%s: set tenant: %w", errPrefix, err)
	}
	q := newQuerier(sqlcgen.New(tx))
	commit := func() error { return tx.Commit(ctx) }
	rollback := func() error {
		rbErr := tx.Rollback(ctx)
		if errors.Is(rbErr, pgx.ErrTxClosed) {
			return nil
		}
		return rbErr
	}
	return q, commit, rollback, nil
}

func newResultTxFactory(pool *pgxpool.Pool) deployment.ResultTxFactory {
	return func(ctx context.Context, tenantID string) (deployment.ResultQuerier, func() error, func() error, error) {
		return beginTenantTx(ctx, pool, tenantID, func(q *sqlcgen.Queries) deployment.ResultQuerier { return q }, "result tx")
	}
}

func newTimeoutTxFactory(pool *pgxpool.Pool) deployment.TimeoutTxFactory {
	return func(ctx context.Context, tenantID string) (deployment.TimeoutQuerier, func() error, func() error, error) {
		return beginTenantTx(ctx, pool, tenantID, func(q *sqlcgen.Queries) deployment.TimeoutQuerier { return q }, "timeout tx")
	}
}

func newWaveDispatcherTxFactory(pool *pgxpool.Pool) deployment.WaveDispatcherTxFactory {
	return func(ctx context.Context, tenantID string) (deployment.WaveDispatcherQuerier, func() error, func() error, error) {
		return beginTenantTx(ctx, pool, tenantID, func(q *sqlcgen.Queries) deployment.WaveDispatcherQuerier { return q }, "wave dispatcher tx")
	}
}
