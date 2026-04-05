@echo off
setlocal
set "ROOT=%~dp0"

powershell -NoProfile -ExecutionPolicy Bypass -File "%ROOT%update.ps1" %*
exit /b %errorlevel%
