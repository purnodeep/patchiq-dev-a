package agent

import (
	"context"
	"time"
)

// NoopEventEmitter is a no-op implementation of EventEmitter, used by CLI commands and tests.
type NoopEventEmitter struct{}

func (NoopEventEmitter) Emit(_ context.Context, _ string, _ any) error { return nil }

// NoopFileCache is a no-op implementation of FileCache, used by CLI commands and tests.
type NoopFileCache struct{}

func (NoopFileCache) Get(_ context.Context, _ string) ([]byte, error) { return nil, nil }
func (NoopFileCache) Put(_ context.Context, _ string, _ []byte) error { return nil }

// NoopConfigProvider is a no-op implementation of ConfigProvider.
type NoopConfigProvider struct{}

func (NoopConfigProvider) GetString(_ string) string          { return "" }
func (NoopConfigProvider) GetInt(_ string) int                { return 0 }
func (NoopConfigProvider) GetDuration(_ string) time.Duration { return 0 }
