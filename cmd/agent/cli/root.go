package cli

import (
	"fmt"
	"os"
)

const (
	ExitOK              = 0
	ExitError           = 1
	ExitConnectionError = 2
)

func Usage() {
	fmt.Fprintf(os.Stderr, `patchiq-agent — PatchIQ endpoint agent

Usage:
  patchiq-agent                  Start the agent daemon
  patchiq-agent install          Guided first-run setup
  patchiq-agent status           Display agent health and connection state
  patchiq-agent scan             Trigger an immediate inventory scan
  patchiq-agent service          Manage Windows service lifecycle
  patchiq-agent <command> --help Show help for a command
`)
}
