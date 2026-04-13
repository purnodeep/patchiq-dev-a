#Requires -RunAsAdministrator
<#
.SYNOPSIS
    PatchIQ Agent Uninstaller for Windows

.DESCRIPTION
    Stops and removes the PatchIQ agent Windows service, binary, and configuration.

.PARAMETER RemoveData
    Optional. Also remove agent data directory (C:\ProgramData\PatchIQ).

.PARAMETER DryRun
    Optional. Print actions without executing.

.EXAMPLE
    .\uninstall-agent-windows.ps1

.EXAMPLE
    .\uninstall-agent-windows.ps1 -RemoveData
#>

[CmdletBinding()]
param(
    [Parameter()]
    [switch]$RemoveData,

    [Parameter()]
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$ScriptVersion = "1.0.0"
$BinaryName = "patchiq-agent.exe"
$InstallDir = "C:\Program Files\PatchIQ"
$ConfigDir = "C:\ProgramData\PatchIQ\config"
$DataDir = "C:\ProgramData\PatchIQ"

function Write-LogInfo  { param([string]$Message) Write-Host "[INFO]  $Message" }
function Write-LogError { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-LogWarn  { param([string]$Message) Write-Host "[WARN]  $Message" -ForegroundColor Yellow }
function Write-LogDry   { param([string]$Message) Write-Host "[DRY-RUN] $Message" -ForegroundColor Cyan }

function Stop-AgentService {
    $binaryPath = Join-Path $InstallDir $BinaryName

    if (-not (Test-Path $binaryPath)) {
        Write-LogWarn "Binary not found at $binaryPath, attempting direct service removal"
        if ($DryRun) {
            Write-LogDry "sc.exe stop PatchIQAgent"
            Write-LogDry "sc.exe delete PatchIQAgent"
            return
        }
        sc.exe stop PatchIQAgent 2>$null
        if ($LASTEXITCODE -notin @(0, 1062)) {
            Write-LogWarn "Failed to stop service (exit code: $LASTEXITCODE)"
        }
        Start-Sleep -Seconds 2
        sc.exe delete PatchIQAgent 2>$null
        if ($LASTEXITCODE -notin @(0, 1060)) {
            Write-LogError "Failed to delete service (exit code: $LASTEXITCODE)"
        }
        return
    }

    Write-LogInfo "Stopping agent service"
    if ($DryRun) {
        Write-LogDry "$binaryPath service stop"
        Write-LogDry "$binaryPath service uninstall"
        return
    }

    & $binaryPath service stop 2>$null
    if ($LASTEXITCODE -notin @(0, 1)) {
        Write-LogWarn "Service stop returned exit code $LASTEXITCODE"
    }
    Start-Sleep -Seconds 2
    & $binaryPath service uninstall
    if ($LASTEXITCODE -ne 0) {
        Write-LogWarn "Service uninstall returned exit code $LASTEXITCODE"
    }
}

function Remove-Binary {
    if (Test-Path $InstallDir) {
        Write-LogInfo "Removing install directory: $InstallDir"
        if ($DryRun) {
            Write-LogDry "Remove-Item -Recurse $InstallDir"
        }
        else {
            Remove-Item -Path $InstallDir -Recurse -Force
        }
    }
    else {
        Write-LogInfo "Install directory not found: $InstallDir"
    }
}

function Remove-FromPath {
    if ($DryRun) {
        Write-LogDry "Remove $InstallDir from system PATH"
        return
    }

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $newPath = ($currentPath -split ";" | Where-Object { $_ -ne $InstallDir }) -join ";"
    if ($currentPath -ne $newPath) {
        [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
        Write-LogInfo "Removed $InstallDir from system PATH"
    }
}

function Remove-Config {
    if (Test-Path $ConfigDir) {
        Write-LogInfo "Removing config directory: $ConfigDir"
        if ($DryRun) {
            Write-LogDry "Remove-Item -Recurse $ConfigDir"
        }
        else {
            Remove-Item -Path $ConfigDir -Recurse -Force
        }
    }
    else {
        Write-LogInfo "Config directory not found: $ConfigDir"
    }
}

function Remove-Data {
    if (-not $RemoveData) {
        Write-LogInfo "Keeping data directory: $DataDir (use -RemoveData to delete)"
        return
    }

    if (Test-Path $DataDir) {
        Write-LogInfo "Removing data directory: $DataDir"
        if ($DryRun) {
            Write-LogDry "Remove-Item -Recurse $DataDir"
        }
        else {
            Remove-Item -Path $DataDir -Recurse -Force
        }
    }
}

# Main
Write-LogInfo "PatchIQ Agent Uninstaller for Windows v$ScriptVersion"

Stop-AgentService
Remove-Binary
Remove-FromPath
Remove-Config
Remove-Data

Write-LogInfo "Uninstall complete"
exit 0
