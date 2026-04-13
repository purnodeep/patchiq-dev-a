package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// OperationalLogWriter writes log entries to the agent's persistent log store.
// This is the same interface as api.LogWriter but redeclared here to avoid
// an import cycle (agent -> api).
type OperationalLogWriter interface {
	WriteLog(ctx context.Context, level, message, source string) error
}

// inventoryCacher is satisfied by inventory.Module when it can provide JSON cache data.
type inventoryCacher interface {
	ExtendedPackagesJSON() ([]byte, error)
}

// InventoryCacheSaver persists inventory cache data to the local store.
type InventoryCacheSaver func(ctx context.Context, data []byte) error

// CollectionRunner periodically runs module Collect() and writes results to the outbox.
type CollectionRunner struct {
	modules      []Module
	outbox       OutboxWriter
	logger       *slog.Logger
	intervalFunc func() time.Duration
	logWriter    OperationalLogWriter
	cacheSaver   InventoryCacheSaver
	collectMu    sync.Mutex // serializes on-demand CollectNow with periodic ticker
}

// NewCollectionRunner creates a runner that periodically collects from modules.
func NewCollectionRunner(modules []Module, outbox OutboxWriter, logger *slog.Logger) *CollectionRunner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &CollectionRunner{modules: modules, outbox: outbox, logger: logger}
}

// SetIntervalFunc sets a function that returns the current scan interval.
// When set, the runner checks this each tick and resets the ticker if the
// interval changed. Must be called before Run.
func (r *CollectionRunner) SetIntervalFunc(f func() time.Duration) {
	r.intervalFunc = f
}

// SetLogWriter sets the persistent log writer for recording operational events.
// Must be called before Run.
func (r *CollectionRunner) SetLogWriter(lw OperationalLogWriter) {
	r.logWriter = lw
}

// SetCacheSaver sets the function used to persist inventory cache data after collection.
func (r *CollectionRunner) SetCacheSaver(fn InventoryCacheSaver) {
	r.cacheSaver = fn
}

// Run starts collection goroutines for each module. Blocks until ctx is cancelled.
func (r *CollectionRunner) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, mod := range r.modules {
		wg.Add(1)
		go func(m Module) {
			defer wg.Done()
			r.runModule(ctx, m)
		}(mod)
	}
	wg.Wait()
}

func (r *CollectionRunner) runModule(ctx context.Context, mod Module) {
	logger := r.logger.With("module", mod.Name())
	interval := mod.CollectInterval()
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	// If a dynamic interval function is set, use it instead of the module default.
	if r.intervalFunc != nil {
		if dynInterval := r.intervalFunc(); dynInterval > 0 {
			interval = dynInterval
		}
	}

	currentInterval := interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	// Run immediately on start.
	r.collectMu.Lock()
	r.collectOnce(ctx, mod, logger)
	r.collectMu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.collectMu.Lock()
			r.collectOnce(ctx, mod, logger)
			r.collectMu.Unlock()
			// Check if interval has changed dynamically.
			if r.intervalFunc != nil {
				newInterval := r.intervalFunc()
				if newInterval > 0 && newInterval != currentInterval {
					logger.Info("collection interval changed", "old", currentInterval, "new", newInterval)
					currentInterval = newInterval
					ticker.Reset(currentInterval)
				}
			}
		}
	}
}

func (r *CollectionRunner) collectOnce(ctx context.Context, mod Module, logger *slog.Logger) {
	items, err := mod.Collect(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "collection failed", "error", err)
		r.writeLog(ctx, "error", fmt.Sprintf("Collection failed for module %s: %v", mod.Name(), err), "collector")
		return
	}

	for _, item := range items {
		if _, err := r.outbox.Add(ctx, item.MessageType, item.Payload); err != nil {
			logger.ErrorContext(ctx, "outbox write failed", "error", err, "message_type", item.MessageType)
		}
	}

	// Cache extended inventory for the local API.
	if r.cacheSaver != nil {
		if cacher, ok := mod.(inventoryCacher); ok {
			if data, err := cacher.ExtendedPackagesJSON(); err != nil {
				logger.WarnContext(ctx, "marshal inventory cache", "error", err)
			} else if data != nil {
				if err := r.cacheSaver(ctx, data); err != nil {
					logger.WarnContext(ctx, "save inventory cache", "error", err)
				}
			}
		}
	}

	r.writeLog(ctx, "info", fmt.Sprintf("Inventory scan completed for module %s, %d items collected", mod.Name(), len(items)), "collector")
}

// CollectNow runs a single collection pass for the named module immediately,
// bypassing the periodic ticker. It is safe to call concurrently with Run —
// a mutex serializes on-demand collection with the periodic ticker so the
// two paths never interleave. Returns an error if the module is not found
// or collection fails.
func (r *CollectionRunner) CollectNow(ctx context.Context, moduleName string) error {
	r.collectMu.Lock()
	defer r.collectMu.Unlock()

	for _, mod := range r.modules {
		if mod.Name() == moduleName {
			logger := r.logger.With("module", mod.Name(), "trigger", "on_demand")
			r.collectOnce(ctx, mod, logger)
			return nil
		}
	}
	return fmt.Errorf("module %q not found in runner", moduleName)
}

// writeLog persists an operational log entry if a log writer is configured.
func (r *CollectionRunner) writeLog(ctx context.Context, level, message, source string) {
	if r.logWriter == nil {
		return
	}
	if err := r.logWriter.WriteLog(ctx, level, message, source); err != nil {
		r.logger.WarnContext(ctx, "write operational log", "error", err)
	}
}
