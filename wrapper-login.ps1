param(
    [string]$Username,
    [string]$Password,
    [switch]$NonInteractive
)

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot
$WRAPPER_DIR = Join-Path $ROOT "wrapper-docker"
$DATA_DIR = Join-Path $WRAPPER_DIR "rootfs\data"

function Ensure-Docker {
    try {
        docker info 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "Docker not running" }
    } catch {
        throw "Docker Desktop is not running"
    }
}

if ([string]::IsNullOrWhiteSpace($Username)) {
    $Username = Read-Host "Apple Music email"
}

if ([string]::IsNullOrWhiteSpace($Password)) {
    $secure = Read-Host "Apple Music password" -AsSecureString
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try {
        $Password = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
    } finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }
}

if ([string]::IsNullOrWhiteSpace($Username) -or [string]::IsNullOrWhiteSpace($Password)) {
    throw "Username and password are required"
}

Ensure-Docker
New-Item -ItemType Directory -Force -Path $DATA_DIR | Out-Null

Write-Host "Starting interactive wrapper login..." -ForegroundColor Cyan

$cachedPath = Join-Path $DATA_DIR "data\com.apple.android.music"

if ($NonInteractive) {
    $existingLogin = docker ps -a --filter "name=^/apple-music-wrapper-login$" --format "{{.ID}}"
    if ($existingLogin) {
        docker rm -f apple-music-wrapper-login 2>$null | Out-Null
    }

    docker run -d `
        -v "${DATA_DIR}:/app/rootfs/data" `
        -e "args=-L ${Username}:${Password} -H 0.0.0.0" `
        --name apple-music-wrapper-login `
        apple-music-wrapper | Out-Null

    if ($LASTEXITCODE -ne 0) {
        throw "Wrapper login failed"
    }

    $completed = $false
    for ($i = 0; $i -lt 60; $i++) {
        Start-Sleep -Seconds 2
        $logs = cmd /c "docker logs apple-music-wrapper-login 2>&1"
        if ((Test-Path $cachedPath) -and $logs -match "account info cached successfully" -and $logs -match "listening 0.0.0.0:10020") {
            $completed = $true
            break
        }
        if ($logs -match "Wrapper login failed") {
            break
        }
    }

    if ($completed) {
        Start-Sleep -Seconds 3
    }

    docker stop apple-music-wrapper-login 2>$null | Out-Null
    docker rm apple-music-wrapper-login 2>$null | Out-Null

    if (-not $completed) {
        throw "Wrapper login timed out before session cache was created"
    }
} else {
    docker run -it --rm `
        -v "${DATA_DIR}:/app/rootfs/data" `
        -e "args=-L ${Username}:${Password} -H 0.0.0.0" `
        --name apple-music-wrapper-login `
        apple-music-wrapper

    if ($LASTEXITCODE -ne 0) {
        throw "Wrapper login failed"
    }
}

$Password = $null
if (-not (Test-Path $cachedPath)) {
    throw "Login finished but no local session cache was created"
}

Write-Host ""
Write-Host "Login flow finished." -ForegroundColor Green
Write-Host "Session cached locally at $cachedPath" -ForegroundColor Green
Write-Host "Run .\wrapper-start.ps1 next." -ForegroundColor Cyan
