//go:build windows

package agent

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sys/windows/svc"
)

// ServiceName is the Windows service name for the PatchIQ agent.
const ServiceName = "PatchIQAgent"

// AgentService implements the Windows service handler (svc.Handler).
type AgentService struct {
	Logger  *slog.Logger
	RunFunc func(ctx context.Context) error
}

// Execute implements svc.Handler. It translates Windows service control signals
// to context cancellation for the agent's run function.
func (s *AgentService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.RunFunc(ctx)
	}()

	changes <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.Logger.Info("windows service stop requested")
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				<-errCh
				return false, 0
			default:
				s.Logger.Warn("unexpected service control request", "cmd", c.Cmd)
			}
		case err := <-errCh:
			if err != nil {
				s.Logger.Error("agent run failed", "error", err)
				return false, 1
			}
			return false, 0
		}
	}
}

// RunAsService starts the agent as a Windows service.
func RunAsService(logger *slog.Logger, runFunc func(ctx context.Context) error) error {
	service := &AgentService{
		Logger:  logger,
		RunFunc: runFunc,
	}
	if err := svc.Run(ServiceName, service); err != nil {
		return fmt.Errorf("run windows service: %w", err)
	}
	return nil
}

// IsWindowsService returns true if the process is running as a Windows service.
func IsWindowsService() bool {
	isWinSvc, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isWinSvc
}
