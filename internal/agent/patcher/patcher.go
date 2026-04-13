package patcher

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/oklog/ulid/v2"
	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent"
	"github.com/skenzeriq/patchiq/internal/agent/store"
	"google.golang.org/protobuf/proto"
)

const defaultCommandTimeout = 30 * time.Minute

// Module implements the patcher agent module.
// sem controls concurrent patch installations (dynamic limit via settings).
// timeout is the max duration for a single install_patch command.
type Module struct {
	logger            *slog.Logger
	installers        map[string]Installer
	executor          CommandExecutor
	sem               *dynamicSem
	timeout           time.Duration
	rollbackStore     *store.RollbackStore
	downloader        *Downloader
	serverHTTPURL     string                                          // base URL for downloading patch binaries from the server
	maxConcurrentFunc func() int                                      // returns current max concurrent installs
	rebootFunc        func(ctx context.Context, delaySec int32) error // optional auto-reboot callback
}

// New creates a new patcher module. Installer is detected at Init time.
func New() *Module {
	return newWithMaxFunc(func() int { return 1 })
}

// NewWithMaxConcurrentFunc creates a patcher module whose concurrency limit
// is read dynamically from maxFunc on every acquire. This allows the
// max_concurrent_installs setting to take effect at runtime.
func NewWithMaxConcurrentFunc(maxFunc func() int) *Module {
	return newWithMaxFunc(maxFunc)
}

func newWithMaxFunc(maxFunc func() int) *Module {
	return &Module{
		sem:               newDynamicSem(maxFunc),
		timeout:           defaultCommandTimeout,
		maxConcurrentFunc: maxFunc,
	}
}

// newTestModule creates a module with injected dependencies for testing.
func newTestModule(inst Installer, exec CommandExecutor) *Module {
	if exec == nil {
		exec = &osExecutor{}
	}
	installers := make(map[string]Installer)
	if inst != nil {
		installers[inst.Name()] = inst
	}
	m := newWithMaxFunc(func() int { return 1 })
	m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	m.installers = installers
	m.executor = exec
	return m
}

func (m *Module) Name() string                   { return "patcher" }
func (m *Module) Version() string                { return "0.1.0" }
func (m *Module) Capabilities() []string         { return []string{"patch_installation"} }
func (m *Module) SupportedCommands() []string    { return []string{"install_patch", "rollback_patch"} }
func (m *Module) CollectInterval() time.Duration { return 0 }

func (m *Module) Init(_ context.Context, deps agent.ModuleDeps) error {
	m.logger = deps.Logger
	if m.logger == nil {
		m.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	m.executor = &osExecutor{}

	if m.installers == nil {
		detected := detectInstallers(defaultInstallerDetectorDeps(), m.executor, m.logger)
		m.installers = make(map[string]Installer, len(detected))
		for _, inst := range detected {
			m.installers[inst.Name()] = inst
			m.logger.Info("patcher installer detected", "installer", inst.Name())
		}
		if len(m.installers) == 0 {
			m.logger.Warn("no package installer detected on this system")
		}
	}

	if deps.LocalDB != nil {
		m.rollbackStore = store.NewRollbackStore(deps.LocalDB)
	}

	if t := deps.ConfigProvider.GetDuration("patcher.command_timeout"); t > 0 {
		m.timeout = t
	}

	// Initialize downloader for fetching patch binaries from the server.
	// server.http_url is the base URL of the Patch Manager HTTP API
	// (e.g. "http://192.168.1.17:8180"). If not set, patch downloads are
	// skipped and install_patch commands rely on the binary being
	// pre-staged on the endpoint.
	if serverURL := deps.ConfigProvider.GetString("server.http_url"); serverURL != "" {
		m.serverHTTPURL = serverURL
		tmpDir := filepath.Join(deps.ConfigProvider.GetString("data_dir"), "patch-downloads")
		if tmpDir == "patch-downloads" {
			// data_dir was empty; use system temp.
			tmpDir = filepath.Join(os.TempDir(), "patchiq-patch-downloads")
		}
		m.downloader = NewDownloader(&http.Client{Timeout: 10 * time.Minute}, tmpDir)
		m.logger.Info("patcher downloader initialized", "server_http_url", serverURL, "tmp_dir", tmpDir)
	}

	return nil
}

func (m *Module) Start(_ context.Context) error       { return nil }
func (m *Module) Stop(_ context.Context) error        { return nil }
func (m *Module) HealthCheck(_ context.Context) error { return nil }

func (m *Module) Collect(_ context.Context) ([]agent.OutboxItem, error) { return nil, nil }

func (m *Module) HandleCommand(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	switch cmd.Type {
	case "install_patch":
		return m.handleInstallPatch(ctx, cmd)
	case "rollback_patch":
		return m.handleRollback(ctx, cmd)
	default:
		return agent.Result{}, fmt.Errorf("patcher: unsupported command type %q", cmd.Type)
	}
}

// preInstallVersion captures the version of a package before installing, keyed by package name.
type preInstallVersion struct {
	packageName string
	fromVersion string
}

func (m *Module) handleInstallPatch(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	// Acquire dynamic semaphore — respects current max_concurrent_installs setting.
	acquireDone := make(chan struct{})
	go func() {
		m.sem.Acquire()
		close(acquireDone)
	}()
	select {
	case <-acquireDone:
		defer m.sem.Release()
	case <-ctx.Done():
		return agent.Result{}, fmt.Errorf("patcher: waiting for lock: %w", ctx.Err())
	}

	// Apply timeout.
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Deserialize payload.
	var payload pb.InstallPatchPayload
	if err := proto.Unmarshal(cmd.Payload, &payload); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("patcher: unmarshal payload: %v", err)}, nil
	}

	// If the server provided a download URL, fetch the binary to a local
	// temp path and override pkg.Name so the installer runs the downloaded
	// file rather than expecting a pre-staged local path.
	if payload.DownloadUrl != "" && m.downloader != nil && m.serverHTTPURL != "" {
		fullURL := m.serverHTTPURL + payload.DownloadUrl
		m.logger.InfoContext(ctx, "patcher: downloading patch binary",
			"command_id", cmd.ID,
			"url", fullURL,
		)
		localPath, err := m.downloader.Download(ctx, fullURL, payload.ChecksumSha256)
		if err != nil {
			return agent.Result{ErrorMessage: fmt.Sprintf("patcher: download binary: %v", err)}, nil
		}
		defer os.Remove(localPath)
		// Override every package's Name with the downloaded local path so
		// the installer uses the fetched binary.
		for _, pkg := range payload.Packages {
			pkg.Name = localPath
		}
		m.logger.InfoContext(ctx, "patcher: binary downloaded",
			"command_id", cmd.ID,
			"local_path", localPath,
		)
	}

	if len(m.installers) == 0 {
		return agent.Result{ErrorMessage: "patcher: no installer available on this system"}, nil
	}

	// Validate packages.
	for _, pkg := range payload.Packages {
		if pkg.Name == "" {
			return agent.Result{ErrorMessage: "patcher: package name is empty"}, nil
		}
	}

	output := &pb.InstallPatchOutput{DryRun: payload.DryRun}

	// Pre-script.
	if payload.PreScript != "" {
		shell, args := scriptShellArgs(payload.PreScript)
		preResult, err := m.executor.Execute(ctx, shell, args...)
		if err != nil {
			return agent.Result{ErrorMessage: fmt.Sprintf("patcher: pre-script execution error: %v", err)}, nil
		}
		output.PreScriptOutput = string(preResult.Stdout) + string(preResult.Stderr)
		if preResult.ExitCode != 0 {
			errMsg := fmt.Sprintf("patcher: pre-script failed with exit code %d", preResult.ExitCode)
			m.logger.WarnContext(ctx, errMsg, "command_id", cmd.ID)
			outputBytes, err := marshalOutput(output)
			if err != nil {
				return agent.Result{}, fmt.Errorf("patcher: command %s: %w", cmd.ID, err)
			}
			return agent.Result{
				Output:       outputBytes,
				ErrorMessage: errMsg,
			}, nil
		}
	}

	// Capture pre-install versions for rollback tracking (non-dry-run only).
	preVersions := make(map[string]string)
	if !payload.DryRun {
		for _, pkg := range payload.Packages {
			inst := m.resolveInstaller(pkg.Source)
			if inst == nil {
				continue
			}
			if vq, ok := inst.(VersionQuerier); ok {
				ver, err := vq.GetCurrentVersion(ctx, pkg.Name)
				if err != nil {
					// Package not installed yet or query failed — no rollback version to record.
					m.logger.DebugContext(ctx, "patcher: could not get pre-install version",
						"package", pkg.Name, "error", err)
					continue
				}
				preVersions[pkg.Name] = ver
			}
		}
	}

	// Install packages.
	var anyFailed bool
	successfulPkgs := make([]preInstallVersion, 0, len(payload.Packages))

	for _, pkg := range payload.Packages {
		target := PatchTarget{Name: pkg.Name, Version: pkg.Version}

		inst := m.resolveInstaller(pkg.Source)
		if inst == nil {
			anyFailed = true
			output.Results = append(output.Results, &pb.InstallResultDetail{
				PackageName: pkg.Name,
				Version:     pkg.Version,
				Stderr:      "patcher: no installer available for package",
				Succeeded:   false,
			})
			continue
		}

		// Pass silent_args from payload to the EXE installer at dispatch time.
		if exeInst, ok := inst.(*exeInstaller); ok && payload.SilentArgs != "" {
			inst = &exeInstaller{
				executor:   exeInst.executor,
				logger:     exeInst.logger,
				silentArgs: payload.SilentArgs,
			}
		}

		result, err := inst.Install(ctx, target, payload.DryRun)

		detail := &pb.InstallResultDetail{
			PackageName:    pkg.Name,
			Version:        pkg.Version,
			ExitCode:       int32(result.ExitCode),
			Stdout:         string(result.Stdout),
			Stderr:         string(result.Stderr),
			RebootRequired: result.RebootRequired,
			Succeeded:      err == nil && result.ExitCode == 0,
		}

		if err != nil {
			detail.Stderr = err.Error()
			anyFailed = true
		} else if result.ExitCode != 0 {
			anyFailed = true
		}

		if detail.Succeeded && !payload.DryRun {
			piv := preInstallVersion{packageName: pkg.Name}
			if v, ok := preVersions[pkg.Name]; ok {
				piv.fromVersion = v
			}
			successfulPkgs = append(successfulPkgs, piv)
		}

		output.Results = append(output.Results, detail)
		m.logger.InfoContext(ctx, "patcher: package install",
			"command_id", cmd.ID,
			"package", pkg.Name,
			"version", pkg.Version,
			"exit_code", result.ExitCode,
			"succeeded", detail.Succeeded,
		)
	}

	// Save rollback records for successfully installed packages.
	if m.rollbackStore != nil && !payload.DryRun && len(successfulPkgs) > 0 {
		for _, sp := range successfulPkgs {
			toVersion := ""
			for _, pkg := range payload.Packages {
				if pkg.Name == sp.packageName {
					toVersion = pkg.Version
					break
				}
			}
			record := &store.RollbackRecord{
				ID:          newULID(),
				CommandID:   cmd.ID,
				PackageName: sp.packageName,
				FromVersion: sp.fromVersion,
				ToVersion:   toVersion,
				Status:      "pending",
			}
			if err := m.rollbackStore.Save(ctx, record); err != nil {
				m.logger.WarnContext(ctx, "patcher: failed to save rollback record",
					"package", sp.packageName, "error", err, "command_id", cmd.ID)
			}
		}
	}

	// Post-script (runs regardless of install outcome).
	if payload.PostScript != "" {
		shell, args := scriptShellArgs(payload.PostScript)
		postResult, err := m.executor.Execute(ctx, shell, args...)
		if err != nil {
			m.logger.WarnContext(ctx, "patcher: post-script execution error", "error", err, "command_id", cmd.ID)
			output.PostScriptOutput = fmt.Sprintf("execution error: %v", err)
			anyFailed = true
		} else {
			output.PostScriptOutput = string(postResult.Stdout) + string(postResult.Stderr)
			if postResult.ExitCode != 0 {
				m.logger.WarnContext(ctx, "patcher: post-script failed", "exit_code", postResult.ExitCode, "command_id", cmd.ID)
				anyFailed = true
			}
		}
	}

	// Auto-reboot: if any installed package requires a reboot and the module
	// has a reboot callback configured, trigger a graceful reboot.
	// The reboot callback is wired by the daemon at startup when auto-reboot
	// is enabled (via settings or future proto field auto_reboot/reboot_delay_seconds).
	if m.rebootFunc != nil {
		var anyRebootRequired bool
		for _, r := range output.Results {
			if r.RebootRequired {
				anyRebootRequired = true
				break
			}
		}
		if anyRebootRequired && !payload.DryRun {
			var delay int32 = 60 // default 60 seconds grace period
			m.logger.InfoContext(ctx, "patcher: auto-reboot triggered",
				"command_id", cmd.ID, "delay_seconds", delay)
			if err := m.rebootFunc(ctx, delay); err != nil {
				m.logger.WarnContext(ctx, "patcher: auto-reboot failed",
					"command_id", cmd.ID, "error", err)
			}
		}
	}

	outputBytes, err := marshalOutput(output)
	if err != nil {
		return agent.Result{}, fmt.Errorf("patcher: command %s: %w", cmd.ID, err)
	}
	agentResult := agent.Result{Output: outputBytes}
	if anyFailed {
		agentResult.ErrorMessage = "patcher: one or more steps failed"
	}

	return agentResult, nil
}

// rollbackPayload is the JSON payload for backward-compatible agent-local rollback.
type rollbackPayload struct {
	CommandID string `json:"command_id"`
}

// handleRollback supports three rollback modes (tried in order):
//  1. Protobuf RollbackPatchPayload with RevertTo targets — server knows correct versions,
//     installs each target directly via the installer. No rollback store needed.
//  2. Protobuf RollbackPatchPayload with OriginalCommandId (no RevertTo) — falls back to
//     local rollback records from the store.
//  3. JSON rollbackPayload (backward compatibility with agent-local rollback).
func (m *Module) handleRollback(ctx context.Context, cmd agent.Command) (agent.Result, error) {
	// Acquire dynamic semaphore — one operation at a time.
	acquireDone := make(chan struct{})
	go func() {
		m.sem.Acquire()
		close(acquireDone)
	}()
	select {
	case <-acquireDone:
		defer m.sem.Release()
	case <-ctx.Done():
		return agent.Result{}, fmt.Errorf("patcher: waiting for lock: %w", ctx.Err())
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Try protobuf first, then fall back to JSON.
	var pbPayload pb.RollbackPatchPayload
	if err := proto.Unmarshal(cmd.Payload, &pbPayload); err == nil {
		// Valid protobuf payload.
		if len(pbPayload.GetRevertTo()) > 0 {
			// Mode 1: Server-specified revert targets — install directly.
			return m.handleRollbackRevertTo(ctx, cmd, &pbPayload)
		}
		if pbPayload.GetOriginalCommandId() != "" {
			// Mode 2: Protobuf with OriginalCommandId — use rollback store.
			return m.handleRollbackFromStore(ctx, cmd, pbPayload.GetOriginalCommandId())
		}
		return agent.Result{ErrorMessage: "patcher: rollback payload missing both revert_to and original_command_id"}, nil
	}

	// Fall back to JSON (backward compatibility).
	var jsonPayload rollbackPayload
	if err := json.Unmarshal(cmd.Payload, &jsonPayload); err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("patcher: unmarshal rollback payload: %v", err)}, nil
	}

	if jsonPayload.CommandID == "" {
		return agent.Result{ErrorMessage: "patcher: rollback payload missing command_id"}, nil
	}

	return m.handleRollbackFromStore(ctx, cmd, jsonPayload.CommandID)
}

// handleRollbackRevertTo installs the server-specified revert targets directly.
func (m *Module) handleRollbackRevertTo(ctx context.Context, cmd agent.Command, payload *pb.RollbackPatchPayload) (agent.Result, error) {
	if len(m.installers) == 0 {
		return agent.Result{ErrorMessage: "patcher: no installer available on this system"}, nil
	}

	output := &pb.InstallPatchOutput{}
	var anyFailed bool

	for _, target := range payload.GetRevertTo() {
		inst := m.resolveInstaller(target.GetSource())
		if inst == nil {
			anyFailed = true
			output.Results = append(output.Results, &pb.InstallResultDetail{
				PackageName: target.GetName(),
				Version:     target.GetVersion(),
				Stderr:      "patcher: no installer available for package",
				Succeeded:   false,
			})
			continue
		}

		pt := PatchTarget{Name: target.GetName(), Version: target.GetVersion()}
		result, installErr := inst.Install(ctx, pt, false)

		detail := &pb.InstallResultDetail{
			PackageName:    target.GetName(),
			Version:        target.GetVersion(),
			ExitCode:       int32(result.ExitCode),
			Stdout:         string(result.Stdout),
			Stderr:         string(result.Stderr),
			RebootRequired: result.RebootRequired,
			Succeeded:      installErr == nil && result.ExitCode == 0,
		}

		if installErr != nil {
			detail.Stderr = installErr.Error()
			anyFailed = true
		} else if result.ExitCode != 0 {
			anyFailed = true
		}

		output.Results = append(output.Results, detail)
		m.logger.InfoContext(ctx, "patcher: rollback revert-to package",
			"command_id", cmd.ID,
			"deployment_id", payload.GetDeploymentId(),
			"package", target.GetName(),
			"version", target.GetVersion(),
			"succeeded", detail.Succeeded,
		)
	}

	outputBytes, err := marshalOutput(output)
	if err != nil {
		return agent.Result{}, fmt.Errorf("patcher: command %s: %w", cmd.ID, err)
	}
	agentResult := agent.Result{Output: outputBytes}
	if anyFailed {
		agentResult.ErrorMessage = "patcher: one or more rollbacks failed"
	}
	return agentResult, nil
}

// handleRollbackFromStore uses local rollback records to downgrade packages.
func (m *Module) handleRollbackFromStore(ctx context.Context, cmd agent.Command, originalCommandID string) (agent.Result, error) {
	if m.rollbackStore == nil {
		return agent.Result{ErrorMessage: "patcher: rollback store not available"}, nil
	}

	records, err := m.rollbackStore.ListByCommand(ctx, originalCommandID)
	if err != nil {
		return agent.Result{ErrorMessage: fmt.Sprintf("patcher: list rollback records: %v", err)}, nil
	}

	if len(records) == 0 {
		return agent.Result{ErrorMessage: fmt.Sprintf("patcher: no rollback records found for command %s", originalCommandID)}, nil
	}

	output := &pb.InstallPatchOutput{}
	var anyFailed bool

	for _, rec := range records {
		if rec.Status != "pending" {
			m.logger.InfoContext(ctx, "patcher: skipping non-pending rollback record",
				"record_id", rec.ID, "status", rec.Status)
			continue
		}

		if rec.FromVersion == "" {
			// Package was not installed before — cannot downgrade, just skip.
			m.logger.InfoContext(ctx, "patcher: no previous version to rollback to",
				"package", rec.PackageName, "command_id", originalCommandID)
			if markErr := m.rollbackStore.MarkFailed(ctx, rec.ID); markErr != nil {
				m.logger.WarnContext(ctx, "patcher: failed to mark rollback record failed",
					"record_id", rec.ID, "error", markErr)
			}
			output.Results = append(output.Results, &pb.InstallResultDetail{
				PackageName: rec.PackageName,
				Version:     rec.FromVersion,
				Stderr:      "no previous version available for rollback",
				Succeeded:   false,
			})
			anyFailed = true
			continue
		}

		inst := m.resolveInstaller("")
		if inst == nil {
			anyFailed = true
			if markErr := m.rollbackStore.MarkFailed(ctx, rec.ID); markErr != nil {
				m.logger.WarnContext(ctx, "patcher: failed to mark rollback record failed",
					"record_id", rec.ID, "error", markErr)
			}
			output.Results = append(output.Results, &pb.InstallResultDetail{
				PackageName: rec.PackageName,
				Stderr:      "patcher: no installer available",
				Succeeded:   false,
			})
			continue
		}

		target := PatchTarget{Name: rec.PackageName, Version: rec.FromVersion}
		result, installErr := inst.Install(ctx, target, false)

		detail := &pb.InstallResultDetail{
			PackageName: rec.PackageName,
			Version:     rec.FromVersion,
			ExitCode:    int32(result.ExitCode),
			Stdout:      string(result.Stdout),
			Stderr:      string(result.Stderr),
			Succeeded:   installErr == nil && result.ExitCode == 0,
		}

		if installErr != nil {
			detail.Stderr = installErr.Error()
		}

		if detail.Succeeded {
			if markErr := m.rollbackStore.MarkCompleted(ctx, rec.ID); markErr != nil {
				m.logger.WarnContext(ctx, "patcher: failed to mark rollback record completed",
					"record_id", rec.ID, "error", markErr)
			}
		} else {
			anyFailed = true
			if markErr := m.rollbackStore.MarkFailed(ctx, rec.ID); markErr != nil {
				m.logger.WarnContext(ctx, "patcher: failed to mark rollback record failed",
					"record_id", rec.ID, "error", markErr)
			}
		}

		output.Results = append(output.Results, detail)
		m.logger.InfoContext(ctx, "patcher: rollback package",
			"command_id", cmd.ID,
			"original_command_id", originalCommandID,
			"package", rec.PackageName,
			"to_version", rec.FromVersion,
			"succeeded", detail.Succeeded,
		)
	}

	outputBytes, err := marshalOutput(output)
	if err != nil {
		return agent.Result{}, fmt.Errorf("patcher: command %s: %w", cmd.ID, err)
	}
	agentResult := agent.Result{Output: outputBytes}
	if anyFailed {
		agentResult.ErrorMessage = "patcher: one or more rollbacks failed"
	}
	return agentResult, nil
}

// resolveInstaller returns the installer for a given source.
// If source is empty, returns the first available installer.
func (m *Module) resolveInstaller(source string) Installer {
	if source != "" {
		return m.installers[source]
	}
	for _, inst := range m.installers {
		return inst
	}
	return nil
}

func marshalOutput(msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("patcher: marshal output: %w", err)
	}
	return data, nil
}

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// Verify Module still satisfies the interface at compile time.
var _ agent.Module = (*Module)(nil)
