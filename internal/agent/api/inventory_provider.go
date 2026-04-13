package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/agent/inventory"
)

// hardwareAdapter implements HardwareProvider by calling the package-level
// inventory.CollectHardware function.
type hardwareAdapter struct {
	logger *slog.Logger
}

// NewHardwareAdapter creates a HardwareProvider that collects hardware info.
func NewHardwareAdapter(logger *slog.Logger) HardwareProvider {
	return &hardwareAdapter{logger: logger}
}

func (a *hardwareAdapter) CollectHardware(ctx context.Context) (*inventory.HardwareInfo, error) {
	return inventory.CollectHardware(ctx, a.logger)
}

// InventoryCacheLoader loads cached inventory JSON bytes from the local store.
// Returns nil, nil when no cache exists.
type InventoryCacheLoader func(ctx context.Context) ([]byte, error)

// softwareAdapter implements SoftwareProvider by aggregating extended packages
// from all collectors in the inventory module.
type softwareAdapter struct {
	module      *inventory.Module
	cacheLoader InventoryCacheLoader
}

// NewSoftwareAdapter creates a SoftwareProvider that reads extended packages
// from the inventory module's collectors, falling back to a cache loader when
// in-memory data is empty (e.g., after a restart).
func NewSoftwareAdapter(module *inventory.Module, cacheLoader InventoryCacheLoader) SoftwareProvider {
	return &softwareAdapter{module: module, cacheLoader: cacheLoader}
}

func (a *softwareAdapter) ExtendedPackages(ctx context.Context) ([]inventory.ExtendedPackageInfo, error) {
	pkgs := a.module.ExtendedPackages()
	if len(pkgs) > 0 {
		return pkgs, nil
	}

	// Fall back to SQLite cache if in-memory is empty (e.g., after restart).
	if a.cacheLoader != nil {
		cached, err := a.cacheLoader(ctx)
		if err != nil {
			return nil, fmt.Errorf("load inventory cache: %w", err)
		}
		if cached != nil {
			var fromCache []inventory.ExtendedPackageInfo
			if err := json.Unmarshal(cached, &fromCache); err != nil {
				return nil, fmt.Errorf("unmarshal inventory cache: %w", err)
			}
			return fromCache, nil
		}
	}

	return nil, nil
}

// servicesAdapter implements ServicesProvider by calling the package-level
// inventory.CollectServices function.
type servicesAdapter struct {
	logger *slog.Logger
}

// NewServicesAdapter creates a ServicesProvider that collects service info.
func NewServicesAdapter(logger *slog.Logger) ServicesProvider {
	return &servicesAdapter{logger: logger}
}

func (a *servicesAdapter) CollectServices(ctx context.Context) ([]inventory.ServiceInfo, error) {
	return inventory.CollectServices(ctx, a.logger)
}
