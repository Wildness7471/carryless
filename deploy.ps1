#Requires -Version 5.1
<#
.SYNOPSIS
    Deploy Carryless on Windows.
.DESCRIPTION
    Installs Go, gcc (via TDM-GCC or winget), SQLite, downloads Go
    module dependencies, builds the binary, and optionally installs a
    Windows service.
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$REQUIRED_GO_MAJOR = 1
$REQUIRED_GO_MINOR = 22
$GO_VERSION        = "1.22.5"
$BINARY_NAME       = "carryless.exe"
$SCRIPT_DIR        = $PSScriptRoot

# ─── Helpers ──────────────────────────────────────────────────────────────────
function Write-Info  { param($msg) Write-Host "[INFO]  $msg" -ForegroundColor Cyan   }
function Write-Ok    { param($msg) Write-Host "[OK]    $msg" -ForegroundColor Green  }
function Write-Warn  { param($msg) Write-Host "[WARN]  $msg" -ForegroundColor Yellow }
function Fail        { param($msg) Write-Host "[ERROR] $msg" -ForegroundColor Red; exit 1 }

function Refresh-Path {
    $env:PATH = [System.Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' +
                [System.Environment]::GetEnvironmentVariable('PATH', 'User')
}

function Test-Command { param($cmd) [bool](Get-Command $cmd -ErrorAction SilentlyContinue) }

function Require-Admin {
    $id = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = [System.Security.Principal.WindowsPrincipal]$id
    if (-not $p.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Fail "This script must be run as Administrator. Right-click PowerShell and choose 'Run as administrator'."
    }
}

# ─── Banner ───────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  Carryless — Windows deployment script"
Write-Host "  ======================================="
Write-Host ""

Require-Admin

# ─── Install winget if missing (Windows 10 1709+) ────────────────────────────
Write-Info "Checking for winget..."
if (-not (Test-Command winget)) {
    Write-Warn "winget not found. Attempting to install via Microsoft Store..."
    Start-Process "ms-windows-store://pdp/?ProductId=9NBLGGH4NNS1" -Wait
    Fail "Please complete the App Installer installation from the Store window, then re-run this script."
}
Write-Ok "winget: $(winget --version)"

# ─── Install / check Go ───────────────────────────────────────────────────────
Write-Info "Checking for Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+..."

function GoVersionOk {
    if (-not (Test-Command go)) { return $false }
    $raw = (go version) -replace '.*go(\d+\.\d+).*','$1'
    $parts = $raw.Split('.')
    $major = [int]$parts[0]; $minor = [int]$parts[1]
    return ($major -gt $REQUIRED_GO_MAJOR) -or ($major -eq $REQUIRED_GO_MAJOR -and $minor -ge $REQUIRED_GO_MINOR)
}

if (-not (GoVersionOk)) {
    Write-Warn "Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+ not found — installing..."
    winget install --id GoLang.Go --version $GO_VERSION --accept-source-agreements --accept-package-agreements --silent
    Refresh-Path
}

if (-not (GoVersionOk)) {
    Fail "Go installation failed. Please install Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+ from https://go.dev/dl/ and re-run."
}
Write-Ok "Go: $(go version)"

# ─── Install GCC (needed for CGO / go-sqlite3) ───────────────────────────────
Write-Info "Checking for gcc (required for CGO)..."

if (-not (Test-Command gcc)) {
    Write-Warn "gcc not found — installing TDM-GCC via winget..."
    # TDM-GCC is the lightest option; MSYS2 is the alternative if this fails.
    $installed = $false
    try {
        winget install --id jmeubank.tdm-gcc --accept-source-agreements --accept-package-agreements --silent
        Refresh-Path
        if (Test-Command gcc) { $installed = $true }
    } catch {}

    if (-not $installed) {
        Write-Warn "TDM-GCC winget install failed. Trying MSYS2..."
        winget install --id MSYS2.MSYS2 --accept-source-agreements --accept-package-agreements --silent
        Refresh-Path
        $msys2 = "C:\msys64\usr\bin"
        if (Test-Path $msys2) {
            [System.Environment]::SetEnvironmentVariable('PATH',
                [System.Environment]::GetEnvironmentVariable('PATH','Machine') + ";$msys2;C:\msys64\mingw64\bin",
                'Machine')
            Refresh-Path
            & C:\msys64\msys2.exe -c "pacman -Sy --noconfirm mingw-w64-x86_64-gcc" | Out-Null
        }
    }
}

if (-not (Test-Command gcc)) {
    Fail "gcc could not be installed automatically.`nInstall TDM-GCC (https://jmeubank.github.io/tdm-gcc/) or MSYS2 mingw64 gcc, add it to PATH, and re-run."
}
Write-Ok "gcc: $(gcc --version | Select-Object -First 1)"

# ─── SQLite (go-sqlite3 bundles its own copy on Windows — nothing extra needed) ─
Write-Info "SQLite: go-sqlite3 bundles SQLite on Windows — no extra install needed."
Write-Ok  "SQLite: OK"

# ─── Download Go module dependencies ─────────────────────────────────────────
Write-Info "Downloading Go module dependencies..."
Set-Location $SCRIPT_DIR
go mod download
go mod verify
Write-Ok "Dependencies downloaded and verified."

# ─── Build ────────────────────────────────────────────────────────────────────
Write-Info "Building ${BINARY_NAME}..."
$env:CGO_ENABLED = '1'
go build -ldflags='-w -s' -o $BINARY_NAME .
Write-Ok "Build complete: ${SCRIPT_DIR}\${BINARY_NAME}"

# ─── Create .env file if missing ─────────────────────────────────────────────
$envFile = Join-Path $SCRIPT_DIR ".env"
if (-not (Test-Path $envFile)) {
    Write-Info "Creating default .env file..."
    @"
# Carryless configuration
# Uncomment and set values as needed.

# PORT=8080
# DATABASE_PATH=carryless.db
# ENVIRONMENT=production   # or: development
# LOG_LEVEL=info

# Optional — Mailgun email notifications
# MAILGUN_DOMAIN=your-domain.com
# MAILGUN_API_KEY=your-api-key
"@ | Set-Content -Path $envFile -Encoding UTF8
    Write-Ok ".env file created at ${envFile}"
} else {
    Write-Info ".env already exists — skipping creation."
}

# ─── Optional: Windows Service ───────────────────────────────────────────────
Write-Host ""
$installService = Read-Host "  Install Carryless as a Windows Service (starts on boot)? [y/N]"
if ($installService -match '^[Yy]$') {
    $svcName = "Carryless"
    $binPath  = Join-Path $SCRIPT_DIR $BINARY_NAME

    # Load .env vars into the service environment string
    $envPairs = @()
    if (Test-Path $envFile) {
        Get-Content $envFile | Where-Object { $_ -notmatch '^\s*#' -and $_ -match '=' } | ForEach-Object {
            $envPairs += $_
        }
    }

    if (Get-Service -Name $svcName -ErrorAction SilentlyContinue) {
        Write-Info "Stopping and removing existing service..."
        Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
        sc.exe delete $svcName | Out-Null
        Start-Sleep -Seconds 2
    }

    Write-Info "Creating Windows service '$svcName'..."
    New-Service -Name $svcName `
                -DisplayName "Carryless Gear Planner" `
                -Description "Carryless backpacking gear planner web application." `
                -BinaryPathName $binPath `
                -StartupType Automatic | Out-Null

    # Set working directory via registry
    $regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$svcName"
    Set-ItemProperty -Path $regPath -Name "AppDirectory" -Value $SCRIPT_DIR -ErrorAction SilentlyContinue

    Start-Service -Name $svcName
    Write-Ok "Service '$svcName' installed and started."
    Write-Ok "Manage it with: Start-Service $svcName  /  Stop-Service $svcName"
} else {
    Write-Host ""
    Write-Info "Skipped service setup."
    Write-Host ""
    Write-Host "  To start Carryless, run:"
    Write-Host "    .\${BINARY_NAME}"
    Write-Host ""
    Write-Host "  Or with a custom port:"
    Write-Host "    `$env:PORT='3000'; .\${BINARY_NAME}"
    Write-Host ""

    $launch = Read-Host "  Start Carryless now? [Y/n]"
    if ($launch -notmatch '^[Nn]$') {
        # Load .env
        if (Test-Path $envFile) {
            Get-Content $envFile | Where-Object { $_ -notmatch '^\s*#' -and $_ -match '=' } | ForEach-Object {
                $kv = $_ -split '=', 2
                [System.Environment]::SetEnvironmentVariable($kv[0].Trim(), $kv[1].Trim(), 'Process')
            }
        }
        Write-Info "Starting Carryless on http://localhost:$($env:PORT ?? '8080') ..."
        & ".\${BINARY_NAME}"
    }
}
