param(
    [string]$InstallDir = "$HOME\.local\bin",
    [switch]$SkipPathUpdate,
    [string]$Version = "",
    [string]$Repo = "vasti/yandex-tracker-cli",
    [switch]$FromRelease,
    [ValidateSet("local", "github", "s3")]
    [string]$Channel = "",
    [string]$BaseUrl = "https://s3.ru-1.storage.selcloud.ru/yandex-tracker-cli"
)

$ErrorActionPreference = 'Stop'

New-Item -ItemType Directory -Force $InstallDir | Out-Null

$output = Join-Path $InstallDir "yt.exe"
$resolvedChannel = $Channel
if (-not $resolvedChannel) {
    if ($FromRelease) {
        $resolvedChannel = "github"
    } else {
        $resolvedChannel = "local"
    }
}

switch ($resolvedChannel) {
    "local" {
        & (Join-Path $PSScriptRoot "build.ps1") -Output $output -Version $Version
    }
    "github" {
        if ($Version) {
            $downloadUrl = "https://github.com/$Repo/releases/download/$Version/yt-windows-amd64.exe"
        } else {
            $downloadUrl = "https://github.com/$Repo/releases/latest/download/yt-windows-amd64.exe"
        }
        Invoke-WebRequest -Uri $downloadUrl -OutFile $output
    }
    "s3" {
        $normalizedBaseUrl = $BaseUrl.TrimEnd('/')
        if ($Version) {
            $downloadUrl = "$normalizedBaseUrl/releases/$Version/yt-windows-amd64.exe"
        } else {
            $downloadUrl = "$normalizedBaseUrl/latest/windows/yt.exe"
        }
        Invoke-WebRequest -Uri $downloadUrl -OutFile $output
    }
    default {
        throw "unsupported install channel: $resolvedChannel"
    }
}

if (-not (Test-Path $output)) {
    throw "installation did not produce $output"
}

if ($resolvedChannel -ne "local") {
    Unblock-File -Path $output -ErrorAction SilentlyContinue
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

Write-Host "Installed yt to $output from channel '$resolvedChannel'"
& $output version
