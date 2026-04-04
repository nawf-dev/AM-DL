@echo off
setlocal
set "ROOT=%~dp0"
set "BIN=%ROOT%amdl.exe"

if not exist "%BIN%" (
  echo amdl.exe not found. Run setup.ps1 first.
  exit /b 1
)

powershell -NoProfile -ExecutionPolicy Bypass -File "%ROOT%wrapper-start.ps1"
if errorlevel 1 exit /b %errorlevel%

"%BIN%"
exit /b %errorlevel%
