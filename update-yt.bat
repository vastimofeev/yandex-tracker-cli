@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "POWERSHELL_EXE=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
set "LOCAL_UPDATE=%SCRIPT_DIR%scripts\update.ps1"
set "REMOTE_UPDATE_URL=https://s3.ru-1.storage.selcloud.ru/yandex-tracker-cli/latest/update.ps1"
set "TEMP_UPDATE=%TEMP%\yt-update.ps1"

echo Updating Yandex Tracker CLI...
if exist "%LOCAL_UPDATE%" (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -File "%LOCAL_UPDATE%" -Channel s3
) else (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri '%REMOTE_UPDATE_URL%' -OutFile '%TEMP_UPDATE%'; & '%TEMP_UPDATE%' -Channel s3"
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
