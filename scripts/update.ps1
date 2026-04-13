param(
    [string]$InstallDir = "$HOME\.local\bin",
    [string]$Version = "",
    [string]$Repo = "vasti/yandex-tracker-cli",
    [switch]$FromRelease,
    [switch]$SkipPathUpdate
)

$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot "install.ps1") `
    -InstallDir $InstallDir `
    -Version $Version `
    -Repo $Repo `
    -FromRelease:$FromRelease `
    -SkipPathUpdate:$SkipPathUpdate
