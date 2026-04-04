# AM-DL

Windows-first Apple Music downloader workspace with a beginner-friendlier CLI flow.

Supported backend target:

- `WorldObservationLog/wrapper`

## Current layout

```text
APPLEMSC_DOWNLOAD/
├── amdl.exe          # Prebuilt Windows binary for quick start
├── client/           # Go source for amdl
├── config.yaml       # Starter config file with safe placeholder values
├── wrapper-docker/   # Docker runtime bundle for wrapper
├── wrapper-src/      # Wrapper source/reference bundle for advanced use
├── tools/            # Optional local binaries like mp4decrypt.exe
├── release/          # Quickstart docs used in release bundle
└── README.md         # Beginner-friendly setup guide
```

## What changed

The client now exposes a cleaner command shell:

```powershell
amdl
amdl setup
amdl doctor
amdl login
amdl logout
amdl token set
amdl search album "Taylor Swift"
amdl download https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
amdl backend status
```

Legacy flags still work, for example:

```powershell
amdl --search album "Taylor Swift"
amdl --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
```

## Prerequisites

Required:

- Docker Desktop
- Go
- MP4Box / GPAC

Optional:

- `ffmpeg`
- `mp4decrypt`

For beginner-friendly MV support, you can place `mp4decrypt.exe` in either:

- the workspace root beside `amdl.exe`
- a local `tools/` folder
- or install it globally in `PATH`

Set the MV / AAC-LC token locally with:

```powershell
.\amdl.exe token set
```

You still need an active Apple Music subscription.

## Quick start for beginners

If `amdl.exe` is already present in the repo or release ZIP, you can skip the manual build step.

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\setup.ps1          # safe to rerun; rebuilds amdl.exe if needed
.\amdl.exe login
.\amdl.exe token set      # optional, needed for MV / AAC-LC
.\wrapper-start.ps1
.\amdl.exe doctor
.\amdl.exe
```

If you want Music Video support, also place `mp4decrypt.exe` in one of these locations:

- beside `amdl.exe`
- in `tools\mp4decrypt.exe`
- anywhere in `PATH`

## Build the client manually

```powershell
cd client
go mod download
go build -o ..\amdl.exe .
```

This produces `amdl.exe` at workspace root.

The repo also includes a starter `config.yaml` with placeholder values, so beginners do not need to create it from scratch.

## Beginner helper scripts

Root helper scripts now exist for the common Windows flow:

- `setup.ps1` — build `amdl.exe`, optionally build wrapper image, and bootstrap root `config.yaml`
- `wrapper-login.ps1` — low-level login helper used by `amdl login`
- `wrapper-start.ps1` — start/stop/status/logs/rebuild for the wrapper container
- `download.bat` — quick download helper that forwards to `amdl.exe`
- `start.bat` — start wrapper, then open the interactive `amdl` menu
- `config.example.yaml` — portable config template for release bundles

Or use `start.bat` after `amdl.exe` is built and login data already exists.

## Advanced: start the wrapper backend manually

Build the Docker image:

```powershell
cd wrapper-docker
docker build --tag apple-music-wrapper .
```

Login once to persist account data:

```powershell
docker run -it --rm `
  -v "${PWD}\rootfs\data:/app/rootfs/data" `
  -e "args=-L YOUR_EMAIL:YOUR_PASSWORD -H 0.0.0.0" `
  --name apple-music-wrapper-login `
  apple-music-wrapper
```

Then start the backend:

```powershell
docker run -d `
  --name apple-music-wrapper `
  --restart unless-stopped `
  --privileged `
  -p 10020:10020 `
  -p 20020:20020 `
  -p 30020:30020 `
  -v "${PWD}\rootfs\data:/app/rootfs/data" `
  -e "args=-H 0.0.0.0" `
  apple-music-wrapper
```

## First-run checklist

1. Run `setup.ps1`
2. Run `amdl.exe login`
3. Optional: run `amdl.exe token set` for MV / AAC-LC
4. Optional: place `mp4decrypt.exe` in `tools/` for MV support
5. Run `wrapper-start.ps1`
6. Run `amdl.exe doctor`
7. Start using `amdl.exe`

## Useful commands

```powershell
.\setup.ps1
.\amdl.exe login
.\amdl.exe logout
.\amdl.exe token set
.\wrapper-start.ps1
.\wrapper-start.ps1 -Status
.\wrapper-start.ps1 -Logs
.\download.bat https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
.\amdl.exe doctor
.\amdl.exe backend status
.\amdl.exe backend guide
.\amdl.exe config show
.\amdl.exe search song "Blinding Lights"
.\amdl.exe download https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b
```

## Notes about wrapper folders

- `wrapper-docker/` is the only runtime folder needed for the beginner flow
- `wrapper-src/` is included for source/reference/advanced fallback usage
- local session/cache lives under `wrapper-docker/rootfs/data` and is not meant to be committed

## Release packaging

This workspace now includes a root GitHub Actions workflow:

- `.github/workflows/release.yml`

It prepares a Windows portable ZIP containing:

- `amdl.exe`
- `config.example.yaml`
- root helper scripts
- `wrapper-docker/`
- quickstart docs

## Included features

- interactive `amdl` menu
- `setup`, `login`, `logout`, `doctor`
- `token set` / `token clear`
- backend status and guide commands
- Windows helper scripts
- release packaging workflow
