//go:build windows

package cli

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "PatchIQAgent"
const serviceDisplayName = "PatchIQ Agent"
const serviceDescription = "PatchIQ endpoint management agent"

// RunService handles the `service` subcommand.
func RunService(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: patchiq-agent service <install|uninstall|start|stop|status|update>")
		return ExitError
	}

	switch args[0] {
	case "install":
		return serviceInstall()
	case "uninstall":
		return serviceUninstall()
	case "start":
		return serviceStart()
	case "stop":
		return serviceStop()
	case "status":
		return serviceStatus()
	case "update":
		return serviceUpdate()
	default:
		fmt.Fprintf(os.Stderr, "unknown service action: %s\n", args[0])
		return ExitError
	}
}

func serviceInstall() int {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get executable path: %v\n", err)
		return ExitError
	}

	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		fmt.Println("service already installed")
		return ExitOK
	}

	s, err = m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName:        serviceDisplayName,
		Description:        serviceDescription,
		StartType:          mgr.StartAutomatic,
		Dependencies: []string{"Winmgmt", "Tcpip"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create service: %v\n", err)
		return ExitError
	}
	defer s.Close()

	if err := s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400); err != nil {
		slog.Error("service install: failed to set recovery actions", "error", err)
		return ExitError
	}

	fmt.Println("service installed successfully")
	return ExitOK
}

func serviceUninstall() int {
	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open service: %v\n", err)
		return ExitError
	}
	defer s.Close()

	s.Control(svc.Stop) //nolint:errcheck
	time.Sleep(2 * time.Second)

	if err := s.Delete(); err != nil {
		fmt.Fprintf(os.Stderr, "delete service: %v\n", err)
		return ExitError
	}

	fmt.Println("service uninstalled successfully")
	return ExitOK
}

func serviceStart() int {
	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open service: %v\n", err)
		return ExitError
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start service: %v\n", err)
		return ExitError
	}

	if err := waitForServiceRunning(serviceName, 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "service health check: %v\n", err)
		return ExitError
	}

	fmt.Println("service started and running")
	return ExitOK
}

func serviceStop() int {
	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open service: %v\n", err)
		return ExitError
	}
	defer s.Close()

	if _, err := s.Control(svc.Stop); err != nil {
		fmt.Fprintf(os.Stderr, "stop service: %v\n", err)
		return ExitError
	}

	fmt.Println("service stopped")
	return ExitOK
}

func serviceStatus() int {
	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Println("service not installed")
		return ExitOK
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		fmt.Fprintf(os.Stderr, "query service: %v\n", err)
		return ExitError
	}

	stateStr := "unknown"
	switch status.State {
	case svc.Stopped:
		stateStr = "stopped"
	case svc.StartPending:
		stateStr = "start_pending"
	case svc.StopPending:
		stateStr = "stop_pending"
	case svc.Running:
		stateStr = "running"
	case svc.ContinuePending:
		stateStr = "continue_pending"
	case svc.PausePending:
		stateStr = "pause_pending"
	case svc.Paused:
		stateStr = "paused"
	}

	fmt.Printf("service: %s\nstate: %s\n", serviceName, stateStr)
	return ExitOK
}

// waitForServiceRunning polls the service status until it reaches the Running
// state or the timeout expires.
func waitForServiceRunning(serviceName string, timeout time.Duration) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	defer s.Close()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("query service: %w", err)
		}
		switch status.State {
		case svc.Running:
			return nil
		case svc.Stopped:
			return fmt.Errorf("service stopped unexpectedly")
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("service did not reach running state within %v", timeout)
}

func serviceUpdate() int {
	// Get the new binary path (current executable).
	newExePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get executable path: %v\n", err)
		return ExitError
	}

	m, err := mgr.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to service manager: %v\n", err)
		return ExitError
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open service: %v\n", err)
		return ExitError
	}

	// Get current config to find the installed binary path.
	cfg, err := s.Config()
	if err != nil {
		s.Close()
		fmt.Fprintf(os.Stderr, "query service config: %v\n", err)
		return ExitError
	}
	installedPath := cfg.BinaryPathName

	// Stop the service.
	status, _ := s.Query()
	if status.State == svc.Running {
		if _, err := s.Control(svc.Stop); err != nil {
			slog.Warn("service update: stop failed, continuing", "error", err)
		}
		time.Sleep(3 * time.Second)
	}
	s.Close()

	// Copy new binary over the installed one (if different paths).
	if newExePath != installedPath {
		src, err := os.ReadFile(newExePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read new binary: %v\n", err)
			return ExitError
		}
		if err := os.WriteFile(installedPath, src, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "write binary to %s: %v\n", installedPath, err)
			return ExitError
		}
		fmt.Printf("binary updated: %s\n", installedPath)
	}

	// Start the service.
	code := serviceStart()
	if code != ExitOK {
		return code
	}

	fmt.Println("service updated successfully")
	return ExitOK
}
