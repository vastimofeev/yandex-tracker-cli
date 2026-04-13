@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "POWERSHELL_EXE=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"

echo Updating Yandex Tracker CLI...
"%POWERSHELL_EXE%" -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%scripts\update.ps1" -FromRelease
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
