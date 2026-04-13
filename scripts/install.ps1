param(
    [string]$InstallDir = "$HOME\.local\bin",
    [switch]$SkipPathUpdate,
    [string]$Version = "",
    [string]$Repo = "vasti/yandex-tracker-cli",
    [switch]$FromRelease
)

$ErrorActionPreference = 'Stop'

New-Item -ItemType Directory -Force $InstallDir | Out-Null

$output = Join-Path $InstallDir "yt.exe"
if ($FromRelease) {
    if ($Version) {
        $downloadUrl = "https://github.com/$Repo/releases/download/$Version/yt-windows-amd64.exe"
    } else {
        $downloadUrl = "https://github.com/$Repo/releases/latest/download/yt-windows-amd64.exe"
    }
    Invoke-WebRequest -Uri $downloadUrl -OutFile $output
} else {
    & (Join-Path $PSScriptRoot "build.ps1") -Output $output -Version $Version
}

if (-not $SkipPathUpdate) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($userPath) {
        $parts = $userPath.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries)
    }
    if ($parts -notcontains $InstallDir) {
        $newPath = (($parts + $InstallDir) | Select-Object -Unique) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "Added $InstallDir to user PATH. Open a new terminal for 'yt' to resolve globally."
    }
}

Write-Host "Installed yt to $output"
& $output version
