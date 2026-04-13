package cli

import "context"

// enrollFunc is the function signature for enrollment, injectable for testing.
// Shared by the Linux (zenity) and Windows (PowerShell WPF) GUI installers.
type enrollFunc func(ctx context.Context, opts installOpts, logStatus func(string)) (string, error)
