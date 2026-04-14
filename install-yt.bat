@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "POWERSHELL_EXE=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
set "LOCAL_INSTALL=%SCRIPT_DIR%scripts\install.ps1"
set "REMOTE_INSTALL_URL=https://s3.ru-1.storage.selcloud.ru/yandex-tracker-cli/latest/install.ps1"
set "TEMP_INSTALL=%TEMP%\yt-install.ps1"

echo Installing Yandex Tracker CLI...
if exist "%LOCAL_INSTALL%" (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -File "%LOCAL_INSTALL%" -Channel s3
) else (
  "%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri '%REMOTE_INSTALL_URL%' -OutFile '%TEMP_INSTALL%'; & '%TEMP_INSTALL%' -Channel s3"
)
set "EXITCODE=%ERRORLEVEL%"

if not "%EXITCODE%"=="0" (
  echo.
  echo Installation failed with exit code %EXITCODE%.
  pause
  exit /b %EXITCODE%
)

echo.
echo Installation completed.
echo Open a new terminal and run: yt version
pause
