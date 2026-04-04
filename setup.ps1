param(
    [switch]$SkipDocker,
    [switch]$SkipConfig,
    [switch]$SkipDoctor,
    [switch]$ForceConfig
)

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot
$CLIENT_DIR = Join-Path $ROOT "client"
$WRAPPER_DIR = Join-Path $ROOT "wrapper-docker"
$BINARY_PATH = Join-Path $ROOT "amdl.exe"
$CONFIG_PATH = Join-Path $ROOT "config.yaml"

function Write-Info($text) { Write-Host "[INFO] $text" -ForegroundColor Cyan }
function Write-Ok($text) { Write-Host "[ OK ] $text" -ForegroundColor Green }
function Write-Warn($text) { Write-Host "[WARN] $text" -ForegroundColor Yellow }
function Write-Err($text) { Write-Host "[FAIL] $text" -ForegroundColor Red }

function Test-CommandExists($name) {
    try {
        Get-Command $name -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

if (-not (Test-Path $CLIENT_DIR)) {
    throw "client directory not found: $CLIENT_DIR"
}

if (-not (Test-CommandExists "go")) {
    throw "Go not found in PATH"
}

if (-not $SkipDocker) {
    if (-not (Test-CommandExists "docker")) {
        throw "Docker not found in PATH"
    }

    Write-Info "Building wrapper Docker image..."
    Push-Location $WRAPPER_DIR
    try {
        docker build --tag apple-music-wrapper .
        if ($LASTEXITCODE -ne 0) {
            throw "Docker build failed"
        }
    } finally {
        Pop-Location
    }
    Write-Ok "Wrapper Docker image ready"
}

Write-Info "Building amdl.exe..."
Push-Location $CLIENT_DIR
try {
    go mod download
    if ($LASTEXITCODE -ne 0) {
        throw "go mod download failed"
    }

    go build -o $BINARY_PATH .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }
} finally {
    Pop-Location
}
Write-Ok "Built $BINARY_PATH"

if ((-not $SkipConfig) -and ((-not (Test-Path $CONFIG_PATH)) -or $ForceConfig)) {
    Write-Info "Creating root config.yaml via amdl config reset..."
    Push-Location $ROOT
    try {
        & $BINARY_PATH config reset
        if ($LASTEXITCODE -ne 0) {
            throw "amdl config reset failed"
        }
    } finally {
        Pop-Location
    }
    Write-Ok "Config ready at $CONFIG_PATH"
} elseif (-not $SkipConfig) {
    Write-Warn "Config already exists at $CONFIG_PATH (use -ForceConfig to overwrite)"
}

if (-not $SkipDoctor) {
    Write-Info "Running amdl doctor..."
    Push-Location $ROOT
    try {
        & $BINARY_PATH doctor
        if ($LASTEXITCODE -ne 0) {
            Write-Warn "Doctor reported issues. Review output above."
        }
    } finally {
        Pop-Location
    }
}

Write-Host ""
Write-Ok "Setup complete"
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. .\amdl.exe login"
Write-Host "  2. .\wrapper-start.ps1"
Write-Host "  3. .\amdl.exe"
