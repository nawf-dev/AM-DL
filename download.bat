@echo off
setlocal
set "ROOT=%~dp0"
set "BIN=%ROOT%amdl.exe"

if not exist "%BIN%" (
  echo amdl.exe not found. Run setup.ps1 first.
  exit /b 1
)

if "%~1"=="" (
  echo Apple Music Downloader helper
  echo.
  echo Examples:
  echo   download.bat https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
  echo   download.bat --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538
  echo   download.bat --search album "Taylor Swift"
  echo.
  echo Opening interactive amdl menu...
  "%BIN%"
  exit /b %errorlevel%
)

"%BIN%" download %*
set "CODE=%errorlevel%"
if not "%CODE%"=="0" (
  echo.
  echo Download failed with exit code %CODE%.
  pause
)
exit /b %CODE%
