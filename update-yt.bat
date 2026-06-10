@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "POWERSHELL_EXE=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
set "LOCAL_UPDATE=%SCRIPT_DIR%scripts\update.ps1"
set "REMOTE_UPDATE_URL=https://raw.githubusercontent.com/vastimofeev/yandex-tracker-cli/main/scripts/update.ps1"
set "TEMP_UPDATE=%TEMP%\yt-update.ps1"

echo Updating Yandex Tracker CLI...
if exist "%LOCAL_UPDATE%" (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -File "%LOCAL_UPDATE%" -FromRelease
) else (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri '%REMOTE_UPDATE_URL%' -OutFile '%TEMP_UPDATE%'; & '%TEMP_UPDATE%' -FromRelease"
)
set "EXITCODE=%ERRORLEVEL%"

if not "%EXITCODE%"=="0" (
  echo.
  echo Update failed with exit code %EXITCODE%.
  pause
  exit /b %EXITCODE%
)

echo.
echo Update completed.
echo Open a new terminal and run: yt version
pause
