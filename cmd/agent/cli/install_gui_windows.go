//go:build windows

package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// psRunner abstracts PowerShell execution for testability.
type psRunner interface {
	RunDialog(script string, outputField string) (string, error)
	ShowMessage(title, text, icon string) error
	ShowProgress(title, text string) (update func(string), close func())
}

// defaultPSRunner implements psRunner using PowerShell with WPF.
type defaultPSRunner struct{}

func (d defaultPSRunner) RunDialog(script string, _ string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass",
		"-Command", script)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		slog.Error("gui dialog failed", "error", err, "stderr", stderr.String())
		return "", err
	}
	// WPF .Add() methods return ints that PowerShell prints to stdout.
	// The actual result is always the LAST line (from Write-Output).
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func (d defaultPSRunner) ShowMessage(title, text, icon string) error {
	script := fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework
[System.Windows.MessageBox]::Show('%s', '%s', 'OK', '%s')
`, escapePS(text), escapePS(title), icon)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass",
		"-Command", script)
	return cmd.Run()
}

func (d defaultPSRunner) ShowProgress(title, text string) (func(string), func()) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework
$w = New-Object System.Windows.Window
$w.Title = '%s'
$w.Width = 560; $w.Height = 300
$w.WindowStartupLocation = 'CenterScreen'
$w.ResizeMode = 'NoResize'
$w.Background = [System.Windows.Media.Brushes]::White

$root = New-Object System.Windows.Controls.DockPanel
$root.LastChildFill = $true

# Top header bar
$topBar = New-Object System.Windows.Controls.Border
$topBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')
$topBar.Padding = '24,16,24,16'
[System.Windows.Controls.DockPanel]::SetDock($topBar, 'Top')
$topTitle = New-Object System.Windows.Controls.TextBlock
$topTitle.Text = 'PatchIQ Agent Setup'
$topTitle.FontSize = 16
$topTitle.FontWeight = 'SemiBold'
$topTitle.Foreground = [System.Windows.Media.Brushes]::White
$topBar.Child = $topTitle
$root.Children.Add($topBar)

# Content
$content = New-Object System.Windows.Controls.StackPanel
$content.Margin = '28,24,28,24'

$lbl = New-Object System.Windows.Controls.TextBlock
$lbl.Text = '%s'
$lbl.FontSize = 13
$lbl.Margin = '0,0,0,16'
$lbl.TextWrapping = 'Wrap'

$pb = New-Object System.Windows.Controls.ProgressBar
$pb.IsIndeterminate = $true
$pb.Height = 6
$pb.Margin = '0,0,0,16'
$pb.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')

$stepsBlock = New-Object System.Windows.Controls.TextBlock
$stepsBlock.Text = ''
$stepsBlock.FontSize = 11
$stepsBlock.Foreground = [System.Windows.Media.Brushes]::Gray
$stepsBlock.TextWrapping = 'Wrap'
$stepsBlock.FontFamily = New-Object System.Windows.Media.FontFamily('Consolas')
$stepsBlock.LineHeight = 20

$content.Children.Add($lbl)
$content.Children.Add($pb)
$content.Children.Add($stepsBlock)
$root.Children.Add($content)

$w.Content = $root
$w.Show()
while ($line = Read-Host) {
    if ($line -eq 'CLOSE') { break }
    if ($line.StartsWith('STEPS:')) {
        $stepsBlock.Text = $line.Substring(6)
    } else {
        $lbl.Text = $line
    }
    [System.Windows.Threading.Dispatcher]::CurrentDispatcher.Invoke([Action]{}, 'Background')
}
$w.Close()
`, escapePS(title), escapePS(text))

	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass",
		"-Command", script)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("gui install: progress dialog stdin pipe failed; falling back to no-op progress", "error", err)
		return func(string) {}, func() {}
	}
	if err := cmd.Start(); err != nil {
		slog.Error("gui install: progress dialog powershell start failed; falling back to no-op progress", "error", err)
		_ = stdin.Close()
		return func(string) {}, func() {}
	}

	update := func(msg string) {
		if _, werr := fmt.Fprintln(stdin, msg); werr != nil {
			slog.Debug("gui install: progress dialog write failed", "error", werr)
		}
	}
	closeFn := func() {
		if _, werr := fmt.Fprintln(stdin, "CLOSE"); werr != nil {
			slog.Debug("gui install: progress dialog close write failed", "error", werr)
		}
		if cerr := stdin.Close(); cerr != nil {
			slog.Debug("gui install: progress dialog stdin close failed", "error", cerr)
		}
		if werr := cmd.Wait(); werr != nil {
			slog.Debug("gui install: progress dialog powershell exit", "error", werr)
		}
	}
	return update, closeFn
}

// escapePS escapes single quotes for PowerShell string literals.
func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// hideConsoleWindow hides the console window so only the GUI dialogs are visible.
func hideConsoleWindow() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		const swHide = 0
		showWindow.Call(hwnd, swHide) //nolint:errcheck
	}
}

// HasGUI returns true on Windows (we use built-in PowerShell WPF).
func HasGUI() bool { return true }

// RunGUIInstall executes the Windows GUI enrollment wizard.
func RunGUIInstall(_ []string) int {
	if !isAdmin() {
		if err := RelaunchAsAdmin(); err != nil {
			slog.Error("gui install: elevation failed", "error", err)
			return ExitError
		}
		return ExitOK
	}
	hideConsoleWindow()

	installer := winGUIInstaller{
		runner: defaultPSRunner{},
		enroll: performEnroll,
	}
	return installer.run()
}

// ShowAlreadyEnrolledDialog displays an info dialog on Windows.
func ShowAlreadyEnrolledDialog() int {
	hideConsoleWindow()
	runner := defaultPSRunner{}
	hostname, _ := os.Hostname()
	_ = runner.ShowMessage("PatchIQ Agent",
		fmt.Sprintf("PatchIQ Agent is already enrolled and running on %s.", hostname),
		"Information")
	return ExitOK
}

// HasZenity returns false on Windows — we use PowerShell WPF instead.
func HasZenity() bool { return false }

// winGUIInstaller encapsulates the Windows GUI enrollment flow.
type winGUIInstaller struct {
	runner psRunner
	enroll enrollFunc
}

func (g *winGUIInstaller) run() int {
	exePath, _ := os.Executable()
	defaultServer := readServerTxtWin(exePath)

	const maxAttempts = 3
	for attempt := range maxAttempts {
		// Run the single-window wizard — returns "server|token" or "".
		result, err := g.runWizard(defaultServer)
		if err != nil {
			slog.Error("gui install: wizard error", "error", err)
			return ExitError
		}
		if result == "" {
			return ExitOK // user cancelled
		}

		parts := strings.SplitN(result, "|", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			slog.Error("gui install: invalid wizard result", "result", result)
			return ExitError
		}
		server, token := parts[0], parts[1]

		// Show progress in a separate window.
		completedSteps := []string{}
		allSteps := []string{
			"Connect to server",
			"Enroll agent",
			"Write configuration",
			"Install Windows service",
			"Start service",
			"Verify health",
		}

		update, closeProgress := g.runner.ShowProgress("PatchIQ Agent Setup", "Preparing...")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		opts := installOpts{
			server:      server,
			token:       token,
			resetConfig: true,
		}

		agentID, enrollErr := g.enroll(ctx, opts, func(msg string) {
			switch {
			case strings.Contains(msg, "Connecting"):
				update(msg)
				g.updateSteps(update, allSteps, completedSteps, 0)
			case strings.Contains(msg, "Enrolling"):
				completedSteps = append(completedSteps, allSteps[0])
				update(msg)
				g.updateSteps(update, allSteps, completedSteps, 1)
			case strings.Contains(msg, "Writing"):
				completedSteps = append(completedSteps, allSteps[1])
				update(msg)
				g.updateSteps(update, allSteps, completedSteps, 2)
			case strings.Contains(msg, "Installing"):
				completedSteps = append(completedSteps, allSteps[2])
				update(msg)
				g.updateSteps(update, allSteps, completedSteps, 3)
			case strings.Contains(msg, "Starting"):
				completedSteps = append(completedSteps, allSteps[3])
				update(msg)
				g.updateSteps(update, allSteps, completedSteps, 4)
			default:
				update(msg)
			}
		})
		cancel()
		closeProgress()

		if enrollErr == nil {
			hostname, _ := os.Hostname()
			g.showSuccess(hostname, agentID, server)
			return ExitOK
		}

		slog.Error("gui install: enrollment failed", "error", enrollErr, "attempt", attempt+1)
		if attempt < maxAttempts-1 {
			g.showRetryError(enrollErr, attempt+1, maxAttempts)
		} else {
			g.showFinalError(enrollErr, maxAttempts)
		}
	}
	return ExitError
}

// runWizard launches a single-window multi-page wizard.
// Returns "server|token" on success, empty string on cancel.
func (g *winGUIInstaller) runWizard(defaultServer string) (string, error) {
	hostname, _ := os.Hostname()
	osInfo := fmt.Sprintf("Windows %s", runtime.GOARCH)
	if defaultServer == "" {
		defaultServer = "localhost:50051"
	}

	script := fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework

# ── Window ──────────────────────────────────────────────────
$w = New-Object System.Windows.Window
$w.Title = 'PatchIQ Agent Setup'
$w.Width = 580; $w.Height = 480
$w.WindowStartupLocation = 'CenterScreen'
$w.ResizeMode = 'NoResize'
$w.Background = [System.Windows.Media.Brushes]::White

$root = New-Object System.Windows.Controls.DockPanel
$root.LastChildFill = $true

# ── Top header bar (persistent) ─────────────────────────────
$topBar = New-Object System.Windows.Controls.Border
$topBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')
$topBar.Padding = '28,18,28,18'
[System.Windows.Controls.DockPanel]::SetDock($topBar, 'Top')

$topStack = New-Object System.Windows.Controls.DockPanel
$topTitle = New-Object System.Windows.Controls.TextBlock
$topTitle.Text = 'PatchIQ Agent Setup'
$topTitle.FontSize = 17
$topTitle.FontWeight = 'SemiBold'
$topTitle.Foreground = [System.Windows.Media.Brushes]::White
$topTitle.VerticalAlignment = 'Center'

$stepLabel = New-Object System.Windows.Controls.TextBlock
$stepLabel.Text = ''
$stepLabel.FontSize = 12
$stepLabel.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#a3bffa')
$stepLabel.VerticalAlignment = 'Center'
$stepLabel.HorizontalAlignment = 'Right'
[System.Windows.Controls.DockPanel]::SetDock($stepLabel, 'Right')

$topStack.Children.Add($stepLabel)
$topStack.Children.Add($topTitle)
$topBar.Child = $topStack
$root.Children.Add($topBar)

# ── Bottom button bar (persistent) ──────────────────────────
$bottomBar = New-Object System.Windows.Controls.Border
$bottomBar.BorderBrush = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#e5e7eb')
$bottomBar.BorderThickness = '0,1,0,0'
$bottomBar.Padding = '28,14,28,14'
$bottomBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f9fafb')
[System.Windows.Controls.DockPanel]::SetDock($bottomBar, 'Bottom')

$btnDock = New-Object System.Windows.Controls.DockPanel

$cancelBtn = New-Object System.Windows.Controls.Button
$cancelBtn.Content = 'Cancel'
$cancelBtn.Width = 90; $cancelBtn.Height = 34; $cancelBtn.FontSize = 13
[System.Windows.Controls.DockPanel]::SetDock($cancelBtn, 'Left')
$cancelBtn.Add_Click({ $w.Tag = ''; $w.Close() })

$nextBtn = New-Object System.Windows.Controls.Button
$nextBtn.Content = 'Get Started'
$nextBtn.Width = 120; $nextBtn.Height = 34; $nextBtn.FontSize = 13
$nextBtn.FontWeight = 'SemiBold'
$nextBtn.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')
$nextBtn.Foreground = [System.Windows.Media.Brushes]::White
$nextBtn.BorderThickness = '0'
$nextBtn.HorizontalAlignment = 'Right'
[System.Windows.Controls.DockPanel]::SetDock($nextBtn, 'Right')

$backBtn = New-Object System.Windows.Controls.Button
$backBtn.Content = [char]0x2190 + ' Back'
$backBtn.Width = 90; $backBtn.Height = 34; $backBtn.FontSize = 13
$backBtn.Visibility = 'Collapsed'
$backBtn.HorizontalAlignment = 'Right'
$backBtn.Margin = '0,0,10,0'
[System.Windows.Controls.DockPanel]::SetDock($backBtn, 'Right')

$btnDock.Children.Add($cancelBtn)
$btnDock.Children.Add($nextBtn)
$btnDock.Children.Add($backBtn)
$bottomBar.Child = $btnDock
$root.Children.Add($bottomBar)

# ── Content area (pages swap here) ──────────────────────────
$contentHost = New-Object System.Windows.Controls.Grid
$root.Children.Add($contentHost)

# ── PAGE 1: Welcome ────────────────────────────────────────
$page1 = New-Object System.Windows.Controls.StackPanel
$page1.Margin = '32,28,32,12'

$p1Header = New-Object System.Windows.Controls.TextBlock
$p1Header.Text = 'Welcome'
$p1Header.FontSize = 20; $p1Header.FontWeight = 'SemiBold'
$p1Header.Margin = '0,0,0,6'
$page1.Children.Add($p1Header)

$p1Sub = New-Object System.Windows.Controls.TextBlock
$p1Sub.Text = 'This wizard will set up the PatchIQ agent on this endpoint.'
$p1Sub.FontSize = 12; $p1Sub.Foreground = [System.Windows.Media.Brushes]::Gray
$p1Sub.Margin = '0,0,0,20'; $p1Sub.TextWrapping = 'Wrap'
$page1.Children.Add($p1Sub)

# System info box
$infoBox = New-Object System.Windows.Controls.Border
$infoBox.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f0f4ff')
$infoBox.CornerRadius = '6'; $infoBox.Padding = '16'; $infoBox.Margin = '0,0,0,20'
$infoInner = New-Object System.Windows.Controls.StackPanel
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Hostname:   %s'; FontSize=12; Margin='0,0,0,3'; FontFamily=(New-Object System.Windows.Media.FontFamily('Consolas'))}))
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Platform:    %s'; FontSize=12; FontFamily=(New-Object System.Windows.Media.FontFamily('Consolas'))}))
$infoBox.Child = $infoInner
$page1.Children.Add($infoBox)

$p1What = New-Object System.Windows.Controls.TextBlock
$p1What.Text = 'What happens next:'; $p1What.FontSize = 13; $p1What.FontWeight = 'SemiBold'; $p1What.Margin = '0,0,0,8'
$page1.Children.Add($p1What)

$bullets = @(
    [char]0x2022 + '  Connect to your Patch Manager server',
    [char]0x2022 + '  Register this endpoint using an enrollment token',
    [char]0x2022 + '  Install PatchIQ as a Windows service',
    [char]0x2022 + '  Start the agent and verify it is healthy'
)
foreach ($b in $bullets) {
    $page1.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text=$b; FontSize=12; Margin='8,0,0,4'}))
}

# ── PAGE 2: Server Address ─────────────────────────────────
$page2 = New-Object System.Windows.Controls.StackPanel
$page2.Margin = '32,28,32,12'
$page2.Visibility = 'Collapsed'

$p2Header = New-Object System.Windows.Controls.TextBlock
$p2Header.Text = 'Server Connection'; $p2Header.FontSize = 20; $p2Header.FontWeight = 'SemiBold'; $p2Header.Margin = '0,0,0,6'
$page2.Children.Add($p2Header)

$p2Desc = New-Object System.Windows.Controls.TextBlock
$p2Desc.Text = 'Enter the address of your PatchIQ Patch Manager server. This is the gRPC endpoint the agent will connect to for enrollment and ongoing management.'
$p2Desc.FontSize = 12; $p2Desc.TextWrapping = 'Wrap'
$p2Desc.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#555')
$p2Desc.Margin = '0,0,0,24'
$page2.Children.Add($p2Desc)

$p2Label = New-Object System.Windows.Controls.TextBlock
$p2Label.Text = 'Server address (host:port)'; $p2Label.FontSize = 12; $p2Label.FontWeight = 'SemiBold'; $p2Label.Margin = '0,0,0,6'
$page2.Children.Add($p2Label)

$serverBox = New-Object System.Windows.Controls.TextBox
$serverBox.Text = '%s'; $serverBox.FontSize = 14; $serverBox.Padding = '8,6,8,6'; $serverBox.Margin = '0,0,0,8'
$page2.Children.Add($serverBox)

$p2Hint = New-Object System.Windows.Controls.TextBlock
$p2Hint.Text = 'Example: 192.168.1.100:50051  or  patchiq.company.com:50051'
$p2Hint.FontSize = 11; $p2Hint.Foreground = [System.Windows.Media.Brushes]::Gray
$page2.Children.Add($p2Hint)

# ── PAGE 3: Enrollment Token ───────────────────────────────
$page3 = New-Object System.Windows.Controls.StackPanel
$page3.Margin = '32,28,32,12'
$page3.Visibility = 'Collapsed'

$p3Header = New-Object System.Windows.Controls.TextBlock
$p3Header.Text = 'Enrollment Token'; $p3Header.FontSize = 20; $p3Header.FontWeight = 'SemiBold'; $p3Header.Margin = '0,0,0,6'
$page3.Children.Add($p3Header)

$p3Desc = New-Object System.Windows.Controls.TextBlock
$p3Desc.Text = 'Paste the enrollment token from your Patch Manager. You can find it in Settings > Agent Downloads, or ask your administrator.'
$p3Desc.FontSize = 12; $p3Desc.TextWrapping = 'Wrap'
$p3Desc.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#555')
$p3Desc.Margin = '0,0,0,24'
$page3.Children.Add($p3Desc)

$p3Label = New-Object System.Windows.Controls.TextBlock
$p3Label.Text = 'Enrollment token'; $p3Label.FontSize = 12; $p3Label.FontWeight = 'SemiBold'; $p3Label.Margin = '0,0,0,6'
$page3.Children.Add($p3Label)

$tokenBox = New-Object System.Windows.Controls.TextBox
$tokenBox.FontSize = 14; $tokenBox.Padding = '8,6,8,6'; $tokenBox.Margin = '0,0,0,8'
$page3.Children.Add($tokenBox)

$p3Hint = New-Object System.Windows.Controls.TextBlock
$p3Hint.Text = 'This is a one-time credential that authorizes this endpoint to register.'
$p3Hint.FontSize = 11; $p3Hint.Foreground = [System.Windows.Media.Brushes]::Gray; $p3Hint.TextWrapping = 'Wrap'
$page3.Children.Add($p3Hint)

# ── PAGE 4: Confirm ────────────────────────────────────────
$page4 = New-Object System.Windows.Controls.StackPanel
$page4.Margin = '32,28,32,12'
$page4.Visibility = 'Collapsed'

$p4Header = New-Object System.Windows.Controls.TextBlock
$p4Header.Text = 'Ready to Enroll'; $p4Header.FontSize = 20; $p4Header.FontWeight = 'SemiBold'; $p4Header.Margin = '0,0,0,6'
$page4.Children.Add($p4Header)

$p4Desc = New-Object System.Windows.Controls.TextBlock
$p4Desc.Text = 'Review the details below, then click Enroll to begin.'
$p4Desc.FontSize = 12; $p4Desc.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#555')
$p4Desc.Margin = '0,0,0,24'; $p4Desc.TextWrapping = 'Wrap'
$page4.Children.Add($p4Desc)

$confirmBox = New-Object System.Windows.Controls.Border
$confirmBox.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f0f4ff')
$confirmBox.CornerRadius = '6'; $confirmBox.Padding = '20'; $confirmBox.Margin = '0,0,0,16'
$confirmInner = New-Object System.Windows.Controls.StackPanel
$confirmEndpoint = New-Object System.Windows.Controls.TextBlock -Property @{FontSize=13; Margin='0,0,0,6'; FontFamily=(New-Object System.Windows.Media.FontFamily('Consolas'))}
$confirmServer   = New-Object System.Windows.Controls.TextBlock -Property @{FontSize=13; Margin='0,0,0,6'; FontFamily=(New-Object System.Windows.Media.FontFamily('Consolas'))}
$confirmToken    = New-Object System.Windows.Controls.TextBlock -Property @{FontSize=13; FontFamily=(New-Object System.Windows.Media.FontFamily('Consolas'))}
$confirmInner.Children.Add($confirmEndpoint)
$confirmInner.Children.Add($confirmServer)
$confirmInner.Children.Add($confirmToken)
$confirmBox.Child = $confirmInner
$page4.Children.Add($confirmBox)

$p4Note = New-Object System.Windows.Controls.TextBlock
$p4Note.Text = 'The agent will be installed as a Windows service and will start automatically on boot.'
$p4Note.FontSize = 11; $p4Note.Foreground = [System.Windows.Media.Brushes]::Gray; $p4Note.TextWrapping = 'Wrap'
$page4.Children.Add($p4Note)

# ── Add all pages to content host ──────────────────────────
$contentHost.Children.Add($page1)
$contentHost.Children.Add($page2)
$contentHost.Children.Add($page3)
$contentHost.Children.Add($page4)

# ── Page navigation logic ──────────────────────────────────
$pages = @($page1, $page2, $page3, $page4)
$currentPage = 0
$stepTexts = @('', 'Step 1 of 2', 'Step 2 of 2', 'Confirm')
$nextTexts = @('Get Started', 'Next  ' + [char]0x2192, 'Next  ' + [char]0x2192, 'Enroll Now')

function Show-Page($idx) {
    for ($i = 0; $i -lt $pages.Count; $i++) {
        if ($i -eq $idx) { $pages[$i].Visibility = 'Visible' }
        else             { $pages[$i].Visibility = 'Collapsed' }
    }
    $stepLabel.Text = $stepTexts[$idx]
    $nextBtn.Content = $nextTexts[$idx]

    if ($idx -eq 0) { $backBtn.Visibility = 'Collapsed' }
    else            { $backBtn.Visibility = 'Visible' }

    # Green enroll button on confirm page
    if ($idx -eq 3) {
        $nextBtn.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#16a34a')
        $nextBtn.Width = 130
        # Populate confirm fields
        $confirmEndpoint.Text = 'Endpoint:  %s'
        $confirmServer.Text   = 'Server:    ' + $serverBox.Text
        $confirmToken.Text    = 'Token:     ' + ($tokenBox.Text.Substring(0, [Math]::Min(8, $tokenBox.Text.Length))) + '...'
    } else {
        $nextBtn.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')
        $nextBtn.Width = 120
    }
}

$nextBtn.Add_Click({
    if ($currentPage -eq 1 -and $serverBox.Text.Trim() -eq '') {
        [System.Windows.MessageBox]::Show('Please enter a server address.', 'PatchIQ Agent Setup', 'OK', 'Warning')
        return
    }
    if ($currentPage -eq 2 -and $tokenBox.Text.Trim() -eq '') {
        [System.Windows.MessageBox]::Show('Please enter an enrollment token.', 'PatchIQ Agent Setup', 'OK', 'Warning')
        return
    }
    if ($currentPage -eq 3) {
        # Final step — output result and close
        $w.Tag = $serverBox.Text.Trim() + '|' + $tokenBox.Text.Trim()
        $w.Close()
        return
    }
    $script:currentPage++
    Show-Page $currentPage
}.GetNewClosure())

$backBtn.Add_Click({
    if ($currentPage -gt 0) {
        $script:currentPage--
        Show-Page $currentPage
    }
}.GetNewClosure())

Show-Page 0

$w.Content = $root
$w.Add_ContentRendered({ $serverBox.Focus() })
$w.ShowDialog() | Out-Null
Write-Output $w.Tag
`, escapePS(hostname), escapePS(osInfo), escapePS(defaultServer), escapePS(hostname))

	return g.runner.RunDialog(script, "")
}

// updateSteps sends a formatted step list to the progress window.
func (g *winGUIInstaller) updateSteps(update func(string), allSteps, completed []string, currentIdx int) {
	var sb strings.Builder
	for i, step := range allSteps {
		if i < len(completed) {
			sb.WriteString(fmt.Sprintf("  [done]  %s\n", step))
		} else if i == currentIdx {
			sb.WriteString(fmt.Sprintf("  [..]    %s\n", step))
		} else {
			sb.WriteString(fmt.Sprintf("  [ ]     %s\n", step))
		}
	}
	update("STEPS:" + sb.String())
}

// showSuccess shows a detailed success dialog.
func (g *winGUIInstaller) showSuccess(hostname, agentID, server string) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework

$w = New-Object System.Windows.Window
$w.Title = 'PatchIQ Agent - Setup Complete'
$w.Width = 560; $w.Height = 400
$w.WindowStartupLocation = 'CenterScreen'
$w.ResizeMode = 'NoResize'
$w.Background = [System.Windows.Media.Brushes]::White

$root = New-Object System.Windows.Controls.DockPanel
$root.LastChildFill = $true

# Top bar (green for success)
$topBar = New-Object System.Windows.Controls.Border
$topBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#16a34a')
$topBar.Padding = '28,18,28,18'
[System.Windows.Controls.DockPanel]::SetDock($topBar, 'Top')
$topTitle = New-Object System.Windows.Controls.TextBlock
$topTitle.Text = [char]0x2713 + '  Setup Complete'
$topTitle.FontSize = 17; $topTitle.FontWeight = 'SemiBold'
$topTitle.Foreground = [System.Windows.Media.Brushes]::White
$topBar.Child = $topTitle
$root.Children.Add($topBar)

# Bottom button
$bottomBar = New-Object System.Windows.Controls.Border
$bottomBar.BorderBrush = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#e5e7eb')
$bottomBar.BorderThickness = '0,1,0,0'
$bottomBar.Padding = '28,14,28,14'
$bottomBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f9fafb')
[System.Windows.Controls.DockPanel]::SetDock($bottomBar, 'Bottom')
$doneBtn = New-Object System.Windows.Controls.Button
$doneBtn.Content = 'Done'; $doneBtn.Width = 100; $doneBtn.Height = 34
$doneBtn.FontSize = 13; $doneBtn.FontWeight = 'SemiBold'
$doneBtn.HorizontalAlignment = 'Right'
$doneBtn.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#16a34a')
$doneBtn.Foreground = [System.Windows.Media.Brushes]::White; $doneBtn.BorderThickness = '0'
$doneBtn.IsDefault = $true
$doneBtn.Add_Click({ $w.Close() })
$bottomBar.Child = $doneBtn
$root.Children.Add($bottomBar)

# Content
$sp = New-Object System.Windows.Controls.StackPanel
$sp.Margin = '32,24,32,12'

$desc = New-Object System.Windows.Controls.TextBlock
$desc.Text = 'PatchIQ Agent has been enrolled and is running as a Windows service. It will start automatically on boot.'
$desc.FontSize = 13; $desc.TextWrapping = 'Wrap'
$desc.Margin = '0,0,0,20'
$sp.Children.Add($desc)

$infoBox = New-Object System.Windows.Controls.Border
$infoBox.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f0fdf4')
$infoBox.CornerRadius = '6'; $infoBox.Padding = '20'
$infoInner = New-Object System.Windows.Controls.StackPanel
$mono = New-Object System.Windows.Media.FontFamily('Consolas')
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Hostname:   %s';  FontSize=13; Margin='0,0,0,5'; FontFamily=$mono}))
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Agent ID:   %s';  FontSize=13; Margin='0,0,0,5'; FontFamily=$mono}))
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Server:     %s';  FontSize=13; Margin='0,0,0,5'; FontFamily=$mono}))
$infoInner.Children.Add((New-Object System.Windows.Controls.TextBlock -Property @{Text='Service:    PatchIQAgent (running)'; FontSize=13; FontFamily=$mono}))
$infoBox.Child = $infoInner
$sp.Children.Add($infoBox)

$root.Children.Add($sp)
$w.Content = $root
$w.ShowDialog() | Out-Null
`, escapePS(hostname), escapePS(agentID), escapePS(server))

	_, _ = g.runner.RunDialog(script, "")
}

// showRetryError shows an error dialog with troubleshooting tips.
func (g *winGUIInstaller) showRetryError(err error, attempt, maxAttempts int) {
	errMsg := err.Error()
	tips := troubleshootingTips(errMsg)

	script := fmt.Sprintf(`
Add-Type -AssemblyName PresentationFramework

$w = New-Object System.Windows.Window
$w.Title = 'PatchIQ Agent - Enrollment Failed'
$w.Width = 560; $w.Height = 400
$w.WindowStartupLocation = 'CenterScreen'
$w.ResizeMode = 'NoResize'
$w.Background = [System.Windows.Media.Brushes]::White

$root = New-Object System.Windows.Controls.DockPanel
$root.LastChildFill = $true

$topBar = New-Object System.Windows.Controls.Border
$topBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#dc2626')
$topBar.Padding = '28,18,28,18'
[System.Windows.Controls.DockPanel]::SetDock($topBar, 'Top')
$topTitle = New-Object System.Windows.Controls.TextBlock
$topTitle.Text = 'Enrollment Failed  (attempt %d of %d)'
$topTitle.FontSize = 16; $topTitle.FontWeight = 'SemiBold'
$topTitle.Foreground = [System.Windows.Media.Brushes]::White
$topBar.Child = $topTitle
$root.Children.Add($topBar)

$bottomBar = New-Object System.Windows.Controls.Border
$bottomBar.BorderBrush = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#e5e7eb')
$bottomBar.BorderThickness = '0,1,0,0'; $bottomBar.Padding = '28,14,28,14'
$bottomBar.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#f9fafb')
[System.Windows.Controls.DockPanel]::SetDock($bottomBar, 'Bottom')
$retryBtn = New-Object System.Windows.Controls.Button
$retryBtn.Content = 'Try Again'; $retryBtn.Width = 110; $retryBtn.Height = 34
$retryBtn.FontSize = 13; $retryBtn.FontWeight = 'SemiBold'; $retryBtn.HorizontalAlignment = 'Right'
$retryBtn.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#1a56db')
$retryBtn.Foreground = [System.Windows.Media.Brushes]::White; $retryBtn.BorderThickness = '0'
$retryBtn.IsDefault = $true; $retryBtn.Add_Click({ $w.Close() })
$bottomBar.Child = $retryBtn
$root.Children.Add($bottomBar)

$sp = New-Object System.Windows.Controls.StackPanel
$sp.Margin = '32,24,32,12'

$errBox = New-Object System.Windows.Controls.Border
$errBox.Background = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#fef2f2')
$errBox.CornerRadius = '6'; $errBox.Padding = '14'; $errBox.Margin = '0,0,0,20'
$errText = New-Object System.Windows.Controls.TextBlock
$errText.Text = '%s'; $errText.FontSize = 11; $errText.TextWrapping = 'Wrap'
$errText.FontFamily = New-Object System.Windows.Media.FontFamily('Consolas')
$errBox.Child = $errText
$sp.Children.Add($errBox)

$tipsHeader = New-Object System.Windows.Controls.TextBlock
$tipsHeader.Text = 'Troubleshooting:'; $tipsHeader.FontSize = 13; $tipsHeader.FontWeight = 'SemiBold'; $tipsHeader.Margin = '0,0,0,8'
$sp.Children.Add($tipsHeader)

$tipsText = New-Object System.Windows.Controls.TextBlock
$tipsText.Text = '%s'; $tipsText.FontSize = 12; $tipsText.TextWrapping = 'Wrap'
$tipsText.Foreground = [System.Windows.Media.BrushConverter]::new().ConvertFrom('#555')
$sp.Children.Add($tipsText)

$root.Children.Add($sp)
$w.Content = $root
$w.ShowDialog() | Out-Null
`, attempt, maxAttempts, escapePS(errMsg), escapePS(tips))

	_, _ = g.runner.RunDialog(script, "")
}

func (g *winGUIInstaller) showFinalError(err error, maxAttempts int) {
	_ = g.runner.ShowMessage("PatchIQ Agent",
		fmt.Sprintf("Enrollment failed after %d attempts.\n\n%v\n\nPlease contact your administrator.", maxAttempts, err),
		"Error")
}

// troubleshootingTips returns context-specific tips.
func troubleshootingTips(errMsg string) string {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "connect") || strings.Contains(lower, "dial"):
		return "- Verify the server address and port\n- Check the Patch Manager is running\n- Check firewall rules"
	case strings.Contains(lower, "token") || strings.Contains(lower, "auth") || strings.Contains(lower, "denied"):
		return "- Verify the enrollment token\n- Tokens may be single-use or expired\n- Generate a new one from the dashboard"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return "- Server may be unreachable\n- Check your network connection\n- Try again in a moment"
	default:
		return "- Verify server address and token\n- Check the Patch Manager is running\n- Contact your administrator"
	}
}

// readServerTxtWin reads server.txt next to the executable.
func readServerTxtWin(exePath string) string {
	dir := filepath.Dir(exePath)
	data, err := os.ReadFile(filepath.Join(dir, "server.txt"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
