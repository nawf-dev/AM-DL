param(
    [switch]$Stop,
    [switch]$Status,
    [switch]$Logs,
    [switch]$Rebuild,
    [switch]$NoPause
)

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot
$WRAPPER_DIR = Join-Path $ROOT "wrapper-docker"
$DATA_DIR = Join-Path $WRAPPER_DIR "rootfs\data"
$SESSION_DIR = Join-Path $DATA_DIR "data\com.apple.android.music"
$CONTAINER = "apple-music-wrapper"

function Pause-IfNeeded {
    if (-not $NoPause) {
        Write-Host ""
        Read-Host "Press Enter to close"
    }
}

function Exit-Script($code) {
    Pause-IfNeeded
    exit $code
}

trap {
    Write-Host ""
    Write-Host $_ -ForegroundColor Red
    Pause-IfNeeded
    exit 1
}

function Ensure-Docker {
    try {
        docker info 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Docker not running" }
    } catch {
        throw "Docker Desktop is not running"
    }
}

function Test-Port($address, $port) {
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $async = $client.BeginConnect($address, $port, $null, $null)
        $ok = $async.AsyncWaitHandle.WaitOne(1000, $false)
        if (-not $ok) { return $false }
        $client.EndConnect($async)
        $client.Close()
        return $true
    } catch {
        return $false
    }
}

Ensure-Docker
New-Item -ItemType Directory -Force -Path $DATA_DIR | Out-Null

if ($Rebuild) {
    Push-Location $WRAPPER_DIR
    try {
        docker build --no-cache --tag apple-music-wrapper .
        if ($LASTEXITCODE -ne 0) { throw "Docker rebuild failed" }
    } finally {
        Pop-Location
    }
    Write-Host "Wrapper image rebuilt." -ForegroundColor Green
    Exit-Script 0
}

if ($Stop) {
    $existing = docker ps -a --filter "name=^/${CONTAINER}$" --format "{{.ID}}"
    if ($existing) {
        docker stop $CONTAINER 2>$null | Out-Null
        docker rm $CONTAINER 2>$null | Out-Null
    }
    Write-Host "Wrapper stopped." -ForegroundColor Green
    Exit-Script 0
}

if ($Logs) {
    docker logs -f $CONTAINER
    Exit-Script $LASTEXITCODE
}

if ($Status) {
    $running = docker ps --filter "name=^/${CONTAINER}$" --format "{{.Status}}"
    if ($running) {
        Write-Host "Wrapper running: $running" -ForegroundColor Green
    } else {
        Write-Host "Wrapper not running" -ForegroundColor Yellow
    }

    foreach ($pair in @(@("Decrypt",10020), @("M3U8",20020), @("Account",30020))) {
        $label = $pair[0]
        $port = [int]$pair[1]
        if (Test-Port "127.0.0.1" $port) {
            Write-Host "  ✅ ${label}: 127.0.0.1:$port" -ForegroundColor Green
        } else {
            Write-Host "  ❌ ${label}: 127.0.0.1:$port" -ForegroundColor Red
        }
    }
    if (Test-Path $SESSION_DIR) {
        Write-Host "  ✅ Session cache: $SESSION_DIR" -ForegroundColor Green
    } else {
        Write-Host "  ⚠ Session cache missing. Run .\wrapper-login.ps1 first." -ForegroundColor Yellow
    }
    Exit-Script 0
}

$runningId = docker ps --filter "name=^/${CONTAINER}$" --format "{{.ID}}"
if ($runningId) {
    Write-Host "Wrapper already running: $runningId" -ForegroundColor Yellow
    Exit-Script 0
}

if (-not (Test-Path $SESSION_DIR)) {
    Write-Host "No local Apple Music session cache detected. Run .\wrapper-login.ps1 first." -ForegroundColor Yellow
}

$stale = docker ps -a --filter "name=^/${CONTAINER}$" --format "{{.ID}}"
if ($stale) {
    docker rm $CONTAINER 2>$null | Out-Null
}

docker run -d `
    --name $CONTAINER `
    --restart unless-stopped `
    --privileged `
    -p 10020:10020 `
    -p 20020:20020 `
    -p 30020:30020 `
    -v "${DATA_DIR}:/app/rootfs/data" `
    -e "args=-H 0.0.0.0" `
    apple-music-wrapper | Out-Null

if ($LASTEXITCODE -ne 0) {
    throw "Failed to start wrapper container"
}

Write-Host "Wrapper started." -ForegroundColor Green
Start-Sleep -Seconds 3
& "$PSCommandPath" -Status -NoPause
Pause-IfNeeded
