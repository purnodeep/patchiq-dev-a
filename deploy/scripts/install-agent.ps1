#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Installs the PatchIQ agent as a Windows service.

.DESCRIPTION
    Downloads the PatchIQ agent binary, enrolls with the Patch Manager server,
    installs as a Windows service (LOCAL SYSTEM), and starts the service.
    Safe to re-run for upgrades.

.PARAMETER ServerAddress
    Patch Manager server address (host:port).

.PARAMETER Token
    Enrollment token for agent registration.

.PARAMETER InstallDir
    Installation directory. Default: C:\Program Files\PatchIQ

.PARAMETER BinaryUrl
    URL to download the agent binary. If not specified, binary must already exist.

.EXAMPLE
    .\install-agent.ps1 -ServerAddress "10.0.0.1:50051" -Token "abc123"
#>
param(
    [Parameter(Mandatory)]
    [string]$ServerAddress,

    [Parameter(Mandatory)]
    [string]$Token,

    [string]$InstallDir = "C:\Program Files\PatchIQ",

    [string]$BinaryUrl
)

$ErrorActionPreference = "Stop"

$agentExe = Join-Path $InstallDir "patchiq-agent.exe"
$serviceName = "PatchIQAgent"

# Stop existing service if running.
$existingService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($existingService -and $existingService.Status -eq "Running") {
    Write-Host "Stopping existing $serviceName service..."
    Stop-Service -Name $serviceName -Force
    Start-Sleep -Seconds 2
}

# Create install directory.
if (-not (Test-Path $InstallDir)) {
    Write-Host "Creating install directory: $InstallDir"
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download binary if URL provided.
if ($BinaryUrl) {
    Write-Host "Downloading agent binary from $BinaryUrl..."
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $BinaryUrl -OutFile $agentExe -UseBasicParsing
    Write-Host "Download complete."
}

if (-not (Test-Path $agentExe)) {
    Write-Error "Agent binary not found at $agentExe. Provide -BinaryUrl or copy binary manually."
    exit 1
}

# Enroll with server.
Write-Host "Enrolling agent with server at $ServerAddress..."
& $agentExe install --server $ServerAddress --token $Token --non-interactive --config (Join-Path $InstallDir "agent.yaml") --data-dir (Join-Path $InstallDir "data")
if ($LASTEXITCODE -ne 0) {
    Write-Error "Agent enrollment failed with exit code $LASTEXITCODE"
    exit 1
}
Write-Host "Enrollment successful."

# Install Windows service.
Write-Host "Installing Windows service..."
& $agentExe service install
if ($LASTEXITCODE -ne 0) {
    Write-Error "Service installation failed with exit code $LASTEXITCODE"
    exit 1
}

# Start service.
Write-Host "Starting service..."
& $agentExe service start
if ($LASTEXITCODE -ne 0) {
    Write-Error "Service start failed with exit code $LASTEXITCODE"
    exit 1
}

# Verify.
Start-Sleep -Seconds 2
$svc = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq "Running") {
    Write-Host "PatchIQ agent installed and running successfully." -ForegroundColor Green
} else {
    Write-Warning "Service installed but may not be running. Check: Get-Service $serviceName"
}
