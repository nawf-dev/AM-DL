param(
    [switch]$SkipDocker,
    [switch]$SkipConfig,
    [switch]$SkipDoctor,
    [switch]$ForceConfig,
    [switch]$NoPause
)

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot
$CLIENT_DIR = Join-Path $ROOT "client"
$WRAPPER_DIR = Join-Path $ROOT "wrapper-docker"
$BINARY_PATH = Join-Path $ROOT "amdl.exe"
$CONFIG_PATH = Join-Path $ROOT "config.yaml"
$VERSION_PATH = Join-Path $ROOT "version.json"

function Write-Info($text) { Write-Host "[INFO] $text" -ForegroundColor Cyan }
function Write-Ok($text) { Write-Host "[ OK ] $text" -ForegroundColor Green }
function Write-Warn($text) { Write-Host "[WARN] $text" -ForegroundColor Yellow }
function Write-Err($text) { Write-Host "[FAIL] $text" -ForegroundColor Red }

function Pause-IfNeeded {
    if (-not $NoPause) {
        Write-Host ""
        Read-Host "Press Enter to close"
    }
}

trap {
    Write-Host ""
    Write-Err $_
    Pause-IfNeeded
    exit 1
}

function Test-CommandExists($name) {
    try {
        Get-Command $name -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

function Test-DockerReady {
    try {
        docker info 2>$null | Out-Null
        return ($LASTEXITCODE -eq 0)
    } catch {
        return $false
    }
}

function Write-VersionMetadata {
    if (-not (Test-CommandExists "git")) {
        return
    }

    Push-Location $ROOT
    try {
        $commit = (& git rev-parse HEAD 2>$null).Trim()
        if (-not $commit) {
            return
        }
        $branch = (& git rev-parse --abbrev-ref HEAD 2>$null).Trim()
        if (-not $branch) {
            $branch = "main"
        }
        $payload = [ordered]@{
            repo        = "nawf-dev/AM-DL"
            branch      = $branch
            commit      = $commit
            shortCommit = $commit.Substring(0, [Math]::Min(7, $commit.Length))
            source      = "git"
            updatedAt   = (Get-Date).ToUniversalTime().ToString("o")
        }
        $payload | ConvertTo-Json | Set-Content -Path $VERSION_PATH -Encoding UTF8
        Write-Ok "Version metadata ready at $VERSION_PATH"
    } finally {
        Pop-Location
    }
}

if (-not (Test-Path $CLIENT_DIR)) {
    throw "client directory not found: $CLIENT_DIR"
}

$hasGo = Test-CommandExists "go"
$hasPrebuiltBinary = Test-Path $BINARY_PATH

if (-not $hasGo -and -not $hasPrebuiltBinary) {
    throw @"
Go not found in PATH, and no prebuilt amdl.exe was found.

You can fix this in one of two ways:
1. Install Go and run .\setup.ps1 again
2. Download/use a repo or release bundle that already includes amdl.exe
"@
}

if (-not $SkipDocker) {
    if (-not (Test-CommandExists "docker")) {
        throw "Docker not found in PATH"
    }

    if (-not (Test-DockerReady)) {
        throw @"
Docker Desktop is not ready.

Please do this first:
1. Open Docker Desktop
2. Wait until it fully finishes starting
3. Make sure it is using Linux containers
4. Run `docker info` and confirm it works
5. Run .\setup.ps1 again
"@
    }

    Write-Info "Building wrapper Docker image..."
    Push-Location $WRAPPER_DIR
    try {
        docker build --tag apple-music-wrapper .
        if ($LASTEXITCODE -ne 0) {
            throw "Docker build failed. Try running `docker build --tag apple-music-wrapper .` inside wrapper-docker to see the full error."
        }
    } finally {
        Pop-Location
    }
    Write-Ok "Wrapper Docker image ready"
}

if ($hasGo) {
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
    Write-VersionMetadata
} else {
    Write-Warn "Go not found in PATH. Using existing prebuilt amdl.exe instead."
}

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
Pause-IfNeeded
