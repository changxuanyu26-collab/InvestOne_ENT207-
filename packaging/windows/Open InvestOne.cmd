@echo off
setlocal EnableExtensions
title InvestOne - Windows
set "ROOT=%~dp0"
set "INVESTONE_BIN=%ROOT%InvestOne.exe"
set "INVESTONE_URL=http://127.0.0.1:18791"
if not exist "%INVESTONE_BIN%" (
  echo InvestOne.exe was not found. Extract the entire ZIP first.
  pause
  exit /b 1
)
if not defined APPDATA (
  echo Your Windows user profile is not configured.
  pause
  exit /b 1
)
if not exist "%APPDATA%\InvestOne\logs" mkdir "%APPDATA%\InvestOne\logs"
powershell.exe -NoProfile -Command "$ErrorActionPreference='Stop'; Start-Process -FilePath $env:INVESTONE_BIN -ArgumentList '-open=true' -WorkingDirectory $env:ROOT -WindowStyle Hidden -RedirectStandardOutput (Join-Path $env:APPDATA 'InvestOne\logs\stdout.log') -RedirectStandardError (Join-Path $env:APPDATA 'InvestOne\logs\stderr.log')"
if errorlevel 1 (
  echo Could not start InvestOne. Review logs in %%APPDATA%%\InvestOne\logs.
  pause
  exit /b 1
)
timeout /t 2 /nobreak >nul
start "" "%INVESTONE_URL%"
echo InvestOne is starting. The browser will open at %INVESTONE_URL%.
echo You can close this window. Settings and data remain in your user profile.
exit /b 0
