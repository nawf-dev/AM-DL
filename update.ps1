param(
    [int]$WaitForPid = 0,
    [string]$ArchivePath = "",
    [switch]$CheckOnly,
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

function Get-VersionFilePath {
    return (Join-Path $ROOT "version.json")
}

function Get-LocalVersionInfo {
    if (Test-CommandExists "git" -and (Test-Path (Join-Path $ROOT ".git"))) {
        Push-Location $ROOT
        try {
            $commit = (& git rev-parse HEAD 2>$null).Trim()
            if ($commit) {
                $branch = (& git rev-parse --abbrev-ref HEAD 2>$null).Trim()
                if (-not $branch) { $branch = $REPO_BRANCH }
                return [ordered]@{
                    repo        = "$REPO_OWNER/$REPO_NAME"
                    branch      = $branch
                    commit      = $commit
                    shortCommit = $commit.Substring(0, [Math]::Min(7, $commit.Length))
                    source      = "git"
                }
            }
        } finally {
            Pop-Location
        }
    }

    $versionPath = Get-VersionFilePath
    if (Test-Path $versionPath) {
        return (Get-Content $versionPath -Raw | ConvertFrom-Json)
    }
    return $null
}

function Get-RemoteVersionInfo {
    $headers = @{
        "Accept" = "application/vnd.github+json"
        "User-Agent" = "AM-DL-Updater"
    }
    $uri = "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/commits/$REPO_BRANCH"
    $response = Invoke-RestMethod -Headers $headers -Uri $uri -UseBasicParsing
    return [ordered]@{
        repo        = "$REPO_OWNER/$REPO_NAME"
        branch      = $REPO_BRANCH
        commit      = $response.sha
        shortCommit = $response.sha.Substring(0, [Math]::Min(7, $response.sha.Length))
        source      = "github"
        updatedAt   = $response.commit.author.date
        url         = $response.html_url
    }
}

function Write-VersionInfo {
    param($VersionInfo)

    if ($null -eq $VersionInfo) { return }
    $VersionInfo | ConvertTo-Json | Set-Content -Path (Get-VersionFilePath) -Encoding UTF8
    Write-Ok "Updated version.json"
}

function Format-Timestamp([string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value)) { return "time unknown" }
    try {
        return ([DateTime]::Parse($Value).ToUniversalTime().ToString("yyyy-MM-dd HH:mm 'UTC'"))
    } catch {
        return $Value
    }
}

function Invoke-UpdateCheck {
    $remote = Get-RemoteVersionInfo
    $local = Get-LocalVersionInfo

    Write-Host "Update source: $REPO_OWNER/$REPO_NAME ($REPO_BRANCH)"
    if ($null -eq $local -or [string]::IsNullOrWhiteSpace($local.commit)) {
        Write-Host "Local version: unknown"
    } else {
        $label = $local.source
        if (-not [string]::IsNullOrWhiteSpace($local.branch)) {
            $label = "$label, $($local.branch)"
        }
        Write-Host "Local version: $($local.shortCommit) ($label)"
    }
    Write-Host "Latest remote: $($remote.shortCommit) ($(Format-Timestamp $remote.updatedAt))"

    if ($null -eq $local -or [string]::IsNullOrWhiteSpace($local.commit)) {
        Write-Warn "Could not determine the current local build exactly. Run .\amdl.exe update to refresh this folder."
        return
    }

    if ($local.commit -eq $remote.commit) {
        Write-Ok "No update available. This folder is already on the latest main build."
        return
    }

    Write-Warn "Update available: $($local.shortCommit) -> $($remote.shortCommit)"
    Write-Host "Run .\amdl.exe update to download and apply the latest files." -ForegroundColor Cyan
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
    if ($CheckOnly) {
        Invoke-UpdateCheck
        Pause-IfNeeded
        exit 0
    }

    Ensure-Directory $tempRoot
    Ensure-Directory $extractRoot

    Wait-ForCaller -PidToWait $WaitForPid

    $remoteVersion = $null
    try {
        $remoteVersion = Get-RemoteVersionInfo
    } catch {
        Write-Warn "Could not fetch remote version metadata before update: $($_.Exception.Message)"
    }

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
        "version.json",
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

    if ($null -ne $remoteVersion) {
        Write-VersionInfo $remoteVersion
    }

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
