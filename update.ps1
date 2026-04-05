param(
    [int]$WaitForPid = 0,
    [string]$ArchivePath = "",
    [switch]$SkipSetup,
    [switch]$NoPause
)

$ErrorActionPreference = "Stop"
$ROOT = $PSScriptRoot
$REPO_OWNER = "nawf-dev"
$REPO_NAME = "AM-DL"
$REPO_BRANCH = "main"

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

function PowerShellBinary {
    if (Test-CommandExists "pwsh") { return "pwsh" }
    return "powershell"
}

function Wait-ForCaller {
    param([int]$PidToWait)

    if ($PidToWait -le 0) { return }

    Write-Info "Waiting for amdl process $PidToWait to exit so files can be replaced..."
    while ($true) {
        $proc = Get-Process -Id $PidToWait -ErrorAction SilentlyContinue
        if ($null -eq $proc) { break }
        Start-Sleep -Milliseconds 300
    }
}

function Ensure-Directory($path) {
    if (-not (Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

function Resolve-SourceRoot($extractRoot) {
    $expected = Join-Path $extractRoot ("$REPO_NAME-$REPO_BRANCH")
    if (Test-Path $expected) {
        return $expected
    }

    $candidates = Get-ChildItem -Path $extractRoot -Directory -ErrorAction SilentlyContinue
    foreach ($candidate in $candidates) {
        if ((Test-Path (Join-Path $candidate.FullName "setup.ps1")) -and (Test-Path (Join-Path $candidate.FullName "client"))) {
            return $candidate.FullName
        }
    }

    throw "Unexpected update archive layout under $extractRoot"
}

function Copy-RootFile {
    param(
        [string]$SourceRoot,
        [string]$RelativePath,
        [switch]$KeepExisting
    )

    $sourcePath = Join-Path $SourceRoot $RelativePath
    if (-not (Test-Path $sourcePath)) {
        Write-Warn "Skipped missing file from update bundle: $RelativePath"
        return
    }

    $destinationPath = Join-Path $ROOT $RelativePath
    $destinationDir = Split-Path $destinationPath -Parent
    if ($destinationDir) {
        Ensure-Directory $destinationDir
    }

    if ($KeepExisting -and (Test-Path $destinationPath)) {
        Write-Warn "Keeping existing $RelativePath"
        return
    }

    Copy-Item $sourcePath $destinationPath -Force
    Write-Ok "Updated $RelativePath"
}

function Invoke-RobocopySync {
    param(
        [string]$Source,
        [string]$Destination,
        [string[]]$ExtraArgs = @()
    )

    if (-not (Test-Path $Source)) {
        Write-Warn "Skipped missing folder from update bundle: $Source"
        return
    }

    Ensure-Directory $Destination
    & robocopy $Source $Destination /E /R:2 /W:1 @ExtraArgs | Out-Host
    if ($LASTEXITCODE -gt 7) {
        throw "robocopy failed for $Source -> $Destination (exit code $LASTEXITCODE)"
    }
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("amdl-update-" + [guid]::NewGuid().ToString("N"))
$downloadArchivePath = Join-Path $tempRoot "amdl-latest.zip"
$extractRoot = Join-Path $tempRoot "extract"

try {
    Ensure-Directory $tempRoot
    Ensure-Directory $extractRoot

    Wait-ForCaller -PidToWait $WaitForPid

    if ([string]::IsNullOrWhiteSpace($ArchivePath)) {
        $archiveUrl = "https://github.com/$REPO_OWNER/$REPO_NAME/archive/refs/heads/$REPO_BRANCH.zip"
        Write-Info "Downloading latest update from $archiveUrl"
        Invoke-WebRequest -Uri $archiveUrl -OutFile $downloadArchivePath -UseBasicParsing
    } else {
        if (-not (Test-Path $ArchivePath)) {
            throw "ArchivePath not found: $ArchivePath"
        }
        Write-Info "Using local update archive: $ArchivePath"
        Copy-Item $ArchivePath $downloadArchivePath -Force
    }

    Write-Info "Extracting update bundle..."
    Expand-Archive -Path $downloadArchivePath -DestinationPath $extractRoot -Force

    $sourceRoot = Resolve-SourceRoot $extractRoot

    Write-Info "Updating root files..."
    foreach ($file in @(
        "amdl.exe",
        "README.md",
        "README-ID.md",
        "setup.ps1",
        "wrapper-login.ps1",
        "wrapper-start.ps1",
        "start.bat",
        "download.bat",
        "update.ps1",
        "update.bat"
    )) {
        Copy-RootFile -SourceRoot $sourceRoot -RelativePath $file
    }
    Copy-RootFile -SourceRoot $sourceRoot -RelativePath "config.yaml" -KeepExisting

    Write-Info "Updating source bundle..."
    Invoke-RobocopySync -Source (Join-Path $sourceRoot "client") -Destination (Join-Path $ROOT "client") -ExtraArgs @("/MIR")
    Invoke-RobocopySync -Source (Join-Path $sourceRoot "wrapper-src") -Destination (Join-Path $ROOT "wrapper-src") -ExtraArgs @("/MIR")
    Invoke-RobocopySync -Source (Join-Path $sourceRoot "wrapper-docker") -Destination (Join-Path $ROOT "wrapper-docker") -ExtraArgs @("/XD", (Join-Path $sourceRoot "wrapper-docker\rootfs\data"), "/XF", ".DS_Store")

    Write-Host ""
    Write-Ok "Latest files copied into this AM-DL folder"

    if (-not $SkipSetup -and (Test-Path (Join-Path $ROOT "setup.ps1"))) {
        if (Test-CommandExists "docker" -and (Test-DockerReady)) {
            Write-Info "Refreshing local setup so the wrapper image matches the updated files..."
            & (PowerShellBinary) -NoProfile -ExecutionPolicy Bypass -File (Join-Path $ROOT "setup.ps1") -SkipConfig -NoPause
            if ($LASTEXITCODE -eq 0) {
                Write-Ok "Local setup refreshed"
            } else {
                Write-Warn "setup.ps1 reported issues. Review the output above and run .\setup.ps1 manually if needed."
            }
        } else {
            Write-Warn "Docker Desktop is not ready. Files are updated, but run .\setup.ps1 later to rebuild the wrapper image."
        }
    } elseif ($SkipSetup) {
        Write-Warn "SkipSetup enabled. Run .\setup.ps1 manually when you want to refresh the wrapper image and checks."
    }

    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. .\wrapper-start.ps1"
    Write-Host "  2. .\amdl.exe doctor"
    Write-Host "  3. .\amdl.exe"
    Pause-IfNeeded
    exit 0
}
finally {
    if (Test-Path $tempRoot) {
        Remove-Item $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
