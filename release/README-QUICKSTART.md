# amdl Windows Quickstart

## 1. Build or download the bundle

If you downloaded a release ZIP, extract it first.

## 2. Run setup

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\setup.ps1
```

This will:

- build `amdl.exe`
- create a default `config.yaml` if needed
- optionally run a doctor check

## 3. Login to wrapper

```powershell
.\amdl.exe login
```

This keeps the password out of `config.yaml` and stores only local session/cache data.

If Apple asks for 2FA, complete it in the terminal session.

## 4. Start wrapper

```powershell
.\wrapper-start.ps1
```

If you want MV / AAC-LC features, set the token locally first:

```powershell
.\amdl.exe token set
```

Check status anytime:

```powershell
.\wrapper-start.ps1 -Status
```

## 5. Start amdl

```powershell
.\amdl.exe
```

Or use the helper:

```powershell
start.bat
```

## Useful commands

```powershell
.\amdl.exe setup
.\amdl.exe login
.\amdl.exe logout
.\amdl.exe token set
.\amdl.exe doctor
.\amdl.exe backend status
.\download.bat https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
```

## Notes

- Backend target is `WorldObservationLog/wrapper`
- Docker Desktop must be running before wrapper commands will work
- `amdl login` stores session/cache locally; it does not write your password into `config.yaml`
- for Music Video downloads, put `mp4decrypt.exe` in `PATH`, beside `amdl.exe`, or inside a `tools/` folder
- `mp4decrypt` is optional unless you need MV downloads
