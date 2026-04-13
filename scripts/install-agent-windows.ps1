#Requires -RunAsAdministrator
<#
.SYNOPSIS
    PatchIQ Agent Installer for Windows

.DESCRIPTION
    Downloads, verifies, and installs the PatchIQ agent as a Windows service.
    Delegates enrollment and service registration to the agent binary's CLI.

.PARAMETER Server
    Required. Patch Manager gRPC address (host:port).

.PARAMETER Token
    Required. One-time enrollment token.

.PARAMETER DownloadURL
    Optional. URL to download the agent binary. If omitted, binary must exist at install path.

.PARAMETER Checksum
    Optional. Expected SHA256 hex digest of the binary.

.PARAMETER InstallDir
    Optional. Install directory. Default: C:\Program Files\PatchIQ

.PARAMETER DryRun
    Optional. Print actions without executing.

.EXAMPLE
    .\install-agent-windows.ps1 -Server "pm.example.com:50051" -Token "abc123"

.EXAMPLE
    .\install-agent-windows.ps1 -Server "pm.example.com:50051" -Token "abc123" -DownloadURL "https://releases.example.com/patchiq-agent.exe"
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Server,

    [Parameter(Mandatory = $true)]
    [string]$Token,

    [Parameter()]
    [string]$DownloadURL,

    [Parameter()]
    [string]$Checksum,

    [Parameter()]
    [string]$InstallDir = "C:\Program Files\PatchIQ",

    [Parameter()]
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$ScriptVersion = "1.0.0"
$BinaryName = "patchiq-agent.exe"
$DataDir = "C:\ProgramData\PatchIQ"
$ConfigDir = "C:\ProgramData\PatchIQ\config"

# Exit codes
$EXIT_OK = 0
$EXIT_ERROR = 1
$EXIT_MISSING_PARAMS = 2
$EXIT_DOWNLOAD_FAILED = 3
$EXIT_CHECKSUM_MISMATCH = 4

function Write-LogInfo  { param([string]$Message) Write-Host "[INFO]  $Message" }
function Write-LogError { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-LogWarn  { param([string]$Message) Write-Host "[WARN]  $Message" -ForegroundColor Yellow }
function Write-LogDry   { param([string]$Message) Write-Host "[DRY-RUN] $Message" -ForegroundColor Cyan }

function Get-AgentBinary {
    param(
        [string]$URL,
        [string]$Destination
    )

    Write-LogInfo "Downloading agent binary from $URL"
    try {
        # Invoke-WebRequest is preferred over WebClient: it respects system proxy settings,
        # enforces TLS certificate validation, and is not deprecated.
        Invoke-WebRequest -Uri $URL -OutFile $Destination -UseBasicParsing
    }
    catch {
        Write-LogError "Download failed: $_"
        exit $EXIT_DOWNLOAD_FAILED
    }
}

function Test-BinaryChecksum {
    param(
        [string]$FilePath,
        [string]$ExpectedChecksum
    )

    if ([string]::IsNullOrEmpty($ExpectedChecksum)) {
        Write-LogWarn "No checksum provided, skipping verification"
        return
    }

    $actualHash = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()
    $expected = $ExpectedChecksum.ToLower()

    if ($actualHash -ne $expected) {
        Write-LogError "Checksum mismatch: expected $expected, got $actualHash"
        exit $EXIT_CHECKSUM_MISMATCH
    }
    Write-LogInfo "Checksum verified: $actualHash"
}

function Get-RemoteChecksum {
    param([string]$URL)

    $checksumURL = "$URL.sha256"
    Write-LogInfo "Fetching checksum from $checksumURL"
    try {
        $response = Invoke-WebRequest -Uri $checksumURL -UseBasicParsing
        return (($response.Content -split '\s+')[0])
    }
    catch {
        Write-LogWarn "Could not fetch checksum from $checksumURL"
        return $null
    }
}

function New-Directories {
    $dirs = @($InstallDir, $DataDir, $ConfigDir)
    foreach ($dir in $dirs) {
        if ($DryRun) {
            Write-LogDry "New-Item -ItemType Directory -Path $dir"
        }
        else {
            if (-not (Test-Path $dir)) {
                New-Item -ItemType Directory -Path $dir -Force | Out-Null
                Write-LogInfo "Created directory: $dir"
            }
        }
    }
}

function Install-Binary {
    $binaryPath = Join-Path $InstallDir $BinaryName

    if (-not [string]::IsNullOrEmpty($DownloadURL)) {
        $tmpFile = [System.IO.Path]::GetTempFileName() + ".exe"

        if ($DryRun) {
            Write-LogDry "Download $DownloadURL -> $tmpFile"
            Write-LogDry "Verify checksum"
            Write-LogDry "Move $tmpFile -> $binaryPath"
            return
        }

        try {
            Get-AgentBinary -URL $DownloadURL -Destination $tmpFile

            # Resolve checksum: explicit param > fetch from URL > skip
            $checksumToVerify = $Checksum
            if ([string]::IsNullOrEmpty($checksumToVerify)) {
                $checksumToVerify = Get-RemoteChecksum -URL $DownloadURL
            }
            Test-BinaryChecksum -FilePath $tmpFile -ExpectedChecksum $checksumToVerify

            Move-Item -Path $tmpFile -Destination $binaryPath -Force
            Write-LogInfo "Binary installed to $binaryPath"
        }
        finally {
            if (Test-Path $tmpFile) { Remove-Item -Path $tmpFile -Force -ErrorAction SilentlyContinue }
        }
    }
    else {
        if (-not (Test-Path $binaryPath)) {
            Write-LogError "Binary not found at $binaryPath and no -DownloadURL provided"
            exit $EXIT_ERROR
        }
        Write-LogInfo "Using existing binary at $binaryPath"
    }
}

function Add-ToPath {
    if ($DryRun) {
        Write-LogDry "Add $InstallDir to system PATH"
        return
    }

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "Machine")
        $env:Path = "$env:Path;$InstallDir"
        Write-LogInfo "Added $InstallDir to system PATH"
    }
    else {
        Write-LogInfo "$InstallDir already in system PATH"
    }
}

function Invoke-AgentEnrollment {
    $binaryPath = Join-Path $InstallDir $BinaryName

    Write-LogInfo "Enrolling agent with server $Server"
    if ($DryRun) {
        Write-LogDry "PATCHIQ_ENROLLMENT_TOKEN=[REDACTED] $binaryPath install --server $Server --non-interactive"
        return
    }

    # Pass token via environment variable to avoid exposing it in the process list
    # (visible via Get-Process, Task Manager, or ETW tracing).
    $env:PATCHIQ_ENROLLMENT_TOKEN = $Token
    try {
        & $binaryPath install --server $Server --non-interactive
        if ($LASTEXITCODE -ne 0) {
            Write-LogError "Agent enrollment failed with exit code $LASTEXITCODE"
            exit $EXIT_ERROR
        }
    }
    finally {
        # Clear the token from the environment immediately after use.
        $env:PATCHIQ_ENROLLMENT_TOKEN = $null
    }
}

function Install-AgentService {
    $binaryPath = Join-Path $InstallDir $BinaryName

    Write-LogInfo "Installing Windows service"
    if ($DryRun) {
        Write-LogDry "$binaryPath service install"
        Write-LogDry "$binaryPath service start"
        return
    }

    & $binaryPath service install
    if ($LASTEXITCODE -ne 0) {
        Write-LogError "Service installation failed with exit code $LASTEXITCODE"
        exit $EXIT_ERROR
    }

    & $binaryPath service start
    if ($LASTEXITCODE -ne 0) {
        Write-LogWarn "Service start failed. Check Event Viewer for details."
    }
}

function Test-Installation {
    $binaryPath = Join-Path $InstallDir $BinaryName

    Write-LogInfo "Verifying installation"
    if ($DryRun) {
        Write-LogDry "$binaryPath service status"
        return
    }

    Start-Sleep -Seconds 2
    & $binaryPath service status
    if ($LASTEXITCODE -ne 0) {
        Write-LogError "Agent service is not running. Check Event Viewer for details."
        exit $EXIT_ERROR
    }
}

function Test-Params {
    # Validate server format: host:port
    if ($Server -notmatch '^[a-zA-Z0-9._-]+:\d+$') {
        Write-LogError "Invalid -Server format '$Server': expected host:port (e.g. pm.example.com:50051)"
        exit $EXIT_MISSING_PARAMS
    }

    # Validate download URL uses HTTPS if provided
    if (-not [string]::IsNullOrEmpty($DownloadURL) -and -not $DownloadURL.StartsWith("https://")) {
        Write-LogError "Refusing to download over insecure connection: -DownloadURL must use https://"
        exit $EXIT_ERROR
    }
}

# Main
Write-LogInfo "PatchIQ Agent Installer for Windows v$ScriptVersion"
Write-LogInfo "Server: $Server"

Test-Params
New-Directories
Install-Binary
Add-ToPath
Invoke-AgentEnrollment
Install-AgentService
Test-Installation

Write-LogInfo "Installation complete"
exit $EXIT_OK
