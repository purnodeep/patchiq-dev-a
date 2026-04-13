package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Module collects endpoint inventory (hostname, OS info, installed packages).
type Module struct {
	logger          *slog.Logger
	collectors      []packageCollector
	lastCollectedAt time.Time
	lastErrors      []string
	mu              sync.Mutex
	outbox          agent.OutboxWriter
	cacheSaver      agent.InventoryCacheSaver
}

// SetOutbox wires the outbox writer used by on-demand run_scan commands so
// they can submit inventory through the same path that periodic scans use.
func (m *Module) SetOutbox(ob agent.OutboxWriter) { m.outbox = ob }

// SetCacheSaver wires the local inventory cache saver used by on-demand
// run_scan commands so the agent UI sees fresh data after a manual scan.
func (m *Module) SetCacheSaver(fn agent.InventoryCacheSaver) { m.cacheSaver = fn }

// New creates a new inventory module.
func New() *Module {
	return &Module{}
}

// newModuleWithCollectors creates a module with pre-configured collectors for testing.
func newModuleWithCollectors(collectors []packageCollector) *Module {
	return &Module{collectors: collectors}
}

func (m *Module) Name() string                   { return "inventory" }
func (m *Module) Version() string                { return "0.2.0" }
func (m *Module) Capabilities() []string         { return []string{"inventory"} }
func (m *Module) SupportedCommands() []string    { return []string{"run_scan"} }
func (m *Module) CollectInterval() time.Duration { return 24 * time.Hour }

func (m *Module) Init(_ context.Context, deps agent.ModuleDeps) error {
	m.logger = deps.Logger
	if m.logger == nil {
		m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	if m.collectors == nil {
		dd := defaultDetectorDeps()
		dd.logger = m.logger
		m.collectors = detectCollectors(dd)
		for _, c := range m.collectors {
			m.logger.Info("inventory collector activated", "collector", c.Name())
		}
		if len(m.collectors) == 0 {
			m.logger.Warn("no package collectors detected on this system")
		}
	}

	return nil
}

// extendedCollector is an optional interface that collectors may implement to
// provide enriched package metadata beyond the gRPC proto fields.
type extendedCollector interface {
	ExtendedPackages() []ExtendedPackageInfo
}

// ExtendedPackagesJSON returns the extended packages as JSON bytes, suitable for caching.
func (m *Module) ExtendedPackagesJSON() ([]byte, error) {
	pkgs := m.ExtendedPackages()
	if len(pkgs) == 0 {
		return nil, nil
	}
	return json.Marshal(pkgs)
}

// ExtendedPackages aggregates extended package info from all collectors that
// support it (e.g., APT and snap collectors).
func (m *Module) ExtendedPackages() []ExtendedPackageInfo {
	var all []ExtendedPackageInfo
	for _, c := range m.collectors {
		if ec, ok := c.(extendedCollector); ok {
			all = append(all, ec.ExtendedPackages()...)
		}
	}
	return all
}

func (m *Module) Start(_ context.Context) error       { return nil }
func (m *Module) Stop(_ context.Context) error        { return nil }
func (m *Module) HealthCheck(_ context.Context) error { return nil }

func (m *Module) Collect(ctx context.Context) ([]agent.OutboxItem, error) {
	report, err := m.buildReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect inventory: %w", err)
	}
	payload, err := proto.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal inventory report: %w", err)
	}
	return []agent.OutboxItem{{MessageType: "inventory", Payload: payload}}, nil
}

func (m *Module) HandleCommand(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	switch cmd.Type {
	case "run_scan":
		m.parseRunScanPayload(cmd.Payload)

		items, err := m.Collect(ctx)
		if err != nil {
			return agent.Result{}, err
		}

		// Submit inventory directly to outbox — same path periodic scans use.
		// Without this the server's command-result handler silently drops the
		// InventoryReport bytes, leaving on-demand scans invisible on the server.
		if m.outbox == nil {
			m.logger.Error("run_scan: outbox not configured — inventory cannot be submitted; check agent wiring")
			return agent.Result{}, fmt.Errorf("run_scan: outbox not configured")
		}
		for _, item := range items {
			if _, err := m.outbox.Add(ctx, item.MessageType, item.Payload); err != nil {
				return agent.Result{}, fmt.Errorf("run_scan: write inventory to outbox: %w", err)
			}
		}

		// Refresh local inventory cache so the agent UI sees fresh data after a
		// manual scan. Cache failures are non-fatal — the submission above is
		// authoritative — but we track them so operators see them in logs.
		if m.cacheSaver != nil {
			if data, err := m.ExtendedPackagesJSON(); err != nil {
				m.logger.Error("run_scan: marshal inventory cache failed", "error", err)
			} else if data != nil {
				if err := m.cacheSaver(ctx, data); err != nil {
					m.logger.Error("run_scan: save inventory cache failed", "error", err)
				}
			}
		}

		return agent.Result{Output: []byte(fmt.Sprintf("scan completed: %d packages", len(items)))}, nil
	default:
		return agent.Result{}, fmt.Errorf("unsupported command type %q", cmd.Type)
	}
}

// parseRunScanPayload attempts to unmarshal a RunScanPayload from the command
// payload. On success it logs the scan type and categories. On failure or empty
// payload it logs a warning and falls through so the caller can continue with a
// full scan (backward compatible).
func (m *Module) parseRunScanPayload(raw []byte) {
	if len(raw) == 0 {
		m.logger.Info("run_scan: no payload, defaulting to full scan")
		return
	}

	var payload pb.RunScanPayload
	if err := proto.Unmarshal(raw, &payload); err != nil {
		m.logger.Warn("run_scan: failed to unmarshal RunScanPayload, defaulting to full scan",
			"error", err)
		return
	}

	m.logger.Info("run_scan: payload received",
		"scan_type", payload.GetScanType().String(),
		"check_categories", payload.GetCheckCategories(),
	)
}

func (m *Module) buildReport(ctx context.Context) (*pb.InventoryReport, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	osFamily := pb.OsFamily_OS_FAMILY_UNSPECIFIED
	switch runtime.GOOS {
	case "linux":
		osFamily = pb.OsFamily_OS_FAMILY_LINUX
	case "windows":
		osFamily = pb.OsFamily_OS_FAMILY_WINDOWS
	case "darwin":
		osFamily = pb.OsFamily_OS_FAMILY_MACOS
	}

	report := &pb.InventoryReport{
		ProtocolVersion: 1,
		EndpointInfo: &pb.EndpointInfo{
			Hostname:  hostname,
			OsFamily:  osFamily,
			OsVersion: runtime.GOOS + "/" + runtime.GOARCH,
			Tags:      make(map[string]string),
		},
		CollectedAt: timestamppb.Now(),
	}

	// Collect hardware info and include as JSON in endpoint tags.
	hwInfo, hwErr := CollectHardware(ctx, m.logger)
	if hwErr != nil {
		m.logger.Warn("hardware collection failed", "error", hwErr)
	} else {
		hwJSON, marshalErr := json.Marshal(hwInfo)
		if marshalErr != nil {
			m.logger.Warn("marshal hardware info", "error", marshalErr)
		} else {
			report.EndpointInfo.Tags["hardware_json"] = string(hwJSON)
		}
		populateEndpointInfoFromHardware(report.EndpointInfo, hwInfo)
	}

	for _, c := range m.collectors {
		pkgs, cerr := c.Collect(ctx)
		if cerr != nil {
			m.logger.Warn("package collector failed", "collector", c.Name(), "error", cerr)
			report.CollectionErrors = append(report.CollectionErrors, &pb.InventoryCollectionError{
				Collector:    c.Name(),
				ErrorMessage: cerr.Error(),
			})
			continue
		}
		report.InstalledPackages = append(report.InstalledPackages, pkgs...)
	}

	m.mu.Lock()
	m.lastCollectedAt = time.Now()
	m.lastErrors = nil
	for _, ce := range report.CollectionErrors {
		m.lastErrors = append(m.lastErrors, ce.Collector+": "+ce.ErrorMessage)
	}
	m.mu.Unlock()

	return report, nil
}

// LastCollectedAt returns when the last inventory collection completed.
func (m *Module) LastCollectedAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCollectedAt
}

// LastErrors returns collector errors from the last collection.
func (m *Module) LastErrors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.lastErrors...)
}

// CollectorNames returns the names of all registered collectors.
func (m *Module) CollectorNames() []string {
	names := make([]string, len(m.collectors))
	for i, c := range m.collectors {
		names[i] = c.Name()
	}
	return names
}

// ScanProgress is sent via the progress callback during buildReport.
type ScanProgress struct {
	Collector      string        // collector name
	Index          int           // 0-based collector index
	Total          int           // total number of collectors
	Phase          string        // "hardware", "collecting", "done"
	Found          int           // packages found so far (cumulative)
	CollectorFound int           // packages found by this collector
	Elapsed        time.Duration // time spent on this collector
	Err            error         // error if collector failed
}

// CollectWithProgress is like Collect but sends progress updates via callback.
func (m *Module) CollectWithProgress(ctx context.Context, onProgress func(ScanProgress)) ([]agent.OutboxItem, error) {
	report, err := m.buildReportWithProgress(ctx, onProgress)
	if err != nil {
		return nil, fmt.Errorf("collect inventory: %w", err)
	}
	payload, err := proto.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal inventory report: %w", err)
	}
	return []agent.OutboxItem{{MessageType: "inventory", Payload: payload}}, nil
}

func (m *Module) buildReportWithProgress(ctx context.Context, onProgress func(ScanProgress)) (*pb.InventoryReport, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	osFamily := pb.OsFamily_OS_FAMILY_UNSPECIFIED
	switch runtime.GOOS {
	case "linux":
		osFamily = pb.OsFamily_OS_FAMILY_LINUX
	case "windows":
		osFamily = pb.OsFamily_OS_FAMILY_WINDOWS
	case "darwin":
		osFamily = pb.OsFamily_OS_FAMILY_MACOS
	}

	report := &pb.InventoryReport{
		ProtocolVersion: 1,
		EndpointInfo: &pb.EndpointInfo{
			Hostname:  hostname,
			OsFamily:  osFamily,
			OsVersion: runtime.GOOS + "/" + runtime.GOARCH,
			Tags:      make(map[string]string),
		},
		CollectedAt: timestamppb.Now(),
	}

	total := len(m.collectors) + 1 // +1 for hardware

	if onProgress != nil {
		onProgress(ScanProgress{Collector: "hardware", Index: 0, Total: total, Phase: "collecting"})
	}
	hwStart := time.Now()
	hwInfo, hwErr := CollectHardware(ctx, m.logger)
	if hwErr != nil {
		m.logger.Warn("hardware collection failed", "error", hwErr)
		if onProgress != nil {
			onProgress(ScanProgress{Collector: "hardware", Index: 0, Total: total, Phase: "done", Elapsed: time.Since(hwStart), Err: hwErr})
		}
	} else {
		hwJSON, marshalErr := json.Marshal(hwInfo)
		if marshalErr != nil {
			m.logger.Warn("marshal hardware info", "error", marshalErr)
		} else {
			report.EndpointInfo.Tags["hardware_json"] = string(hwJSON)
		}
		populateEndpointInfoFromHardware(report.EndpointInfo, hwInfo)
		if onProgress != nil {
			onProgress(ScanProgress{Collector: "hardware", Index: 0, Total: total, Phase: "done", Elapsed: time.Since(hwStart)})
		}
	}

	totalFound := 0
	for i, c := range m.collectors {
		idx := i + 1
		if onProgress != nil {
			onProgress(ScanProgress{Collector: c.Name(), Index: idx, Total: total, Phase: "collecting", Found: totalFound})
		}
		cStart := time.Now()
		pkgs, cerr := c.Collect(ctx)
		elapsed := time.Since(cStart)
		if cerr != nil {
			m.logger.Warn("package collector failed", "collector", c.Name(), "error", cerr)
			report.CollectionErrors = append(report.CollectionErrors, &pb.InventoryCollectionError{
				Collector:    c.Name(),
				ErrorMessage: cerr.Error(),
			})
			if onProgress != nil {
				onProgress(ScanProgress{Collector: c.Name(), Index: idx, Total: total, Phase: "done", Found: totalFound, Elapsed: elapsed, Err: cerr})
			}
			continue
		}
		report.InstalledPackages = append(report.InstalledPackages, pkgs...)
		totalFound += len(pkgs)
		if onProgress != nil {
			onProgress(ScanProgress{Collector: c.Name(), Index: idx, Total: total, Phase: "done", Found: totalFound, CollectorFound: len(pkgs), Elapsed: elapsed})
		}
	}

	m.mu.Lock()
	m.lastCollectedAt = time.Now()
	m.lastErrors = nil
	for _, ce := range report.CollectionErrors {
		m.lastErrors = append(m.lastErrors, ce.Collector+": "+ce.ErrorMessage)
	}
	m.mu.Unlock()

	return report, nil
}

// populateEndpointInfoFromHardware maps collected HardwareInfo fields into the
// first-class EndpointInfo proto fields. It also sets OS version detail via the
// platform-specific collectOSVersionDetail/collectOSVersion helpers.
func populateEndpointInfoFromHardware(info *pb.EndpointInfo, hw *HardwareInfo) {
	// OS version detail (e.g. "macOS 15.2 (24C101)" or "Ubuntu 22.04.3 LTS").
	if detail := collectOSVersionDetail(); detail != "" {
		info.OsVersionDetail = detail
	}

	// Actual OS version (e.g. "15.2" or "22.04.3") instead of GOOS/GOARCH.
	if ver := collectOSVersion(); ver != "" {
		info.OsVersion = ver
	}

	// Hardware model from motherboard info (e.g. "Mac16,13" or "Dell PowerEdge R740").
	if hw.Motherboard.BoardProduct != "" {
		info.HardwareModel = hw.Motherboard.BoardProduct
	}

	// CPU type from collected CPU info (e.g. "Apple M4" or "Intel Xeon E5-2680").
	if hw.CPU.ModelName != "" {
		info.CpuType = hw.CPU.ModelName
	}

	// Total physical memory in bytes.
	if hw.Memory.TotalBytes > 0 {
		info.MemoryBytes = hw.Memory.TotalBytes
	}

	// Kernel version (e.g. "25.3.0" on macOS, "5.15.0-100-generic" on Linux).
	if kv := collectKernelVersion(); kv != "" {
		if info.Tags == nil {
			info.Tags = make(map[string]string)
		}
		info.Tags["kernel_version"] = kv
	}

	// IP addresses from collected network interfaces.
	var ips []string
	for _, nic := range hw.Network {
		for _, addr := range nic.IPv4Addresses {
			if addr.Address != "" {
				ips = append(ips, addr.Address)
			}
		}
	}
	if len(ips) > 0 {
		info.IpAddresses = ips
	}
}
