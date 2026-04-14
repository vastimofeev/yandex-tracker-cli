param(
    [string]$InstallDir = "$HOME\.local\bin",
    [string]$Version = "",
    [string]$Repo = "vasti/yandex-tracker-cli",
    [switch]$FromRelease,
    [switch]$SkipPathUpdate,
    [ValidateSet("local", "github", "s3")]
    [string]$Channel = "",
    [string]$BaseUrl = "https://s3.ru-1.storage.selcloud.ru/yandex-tracker-cli"
)

$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot "install.ps1") `
    -InstallDir $InstallDir `
    -Version $Version `
    -Repo $Repo `
    -FromRelease:$FromRelease `
    -SkipPathUpdate:$SkipPathUpdate `
    -Channel $Channel `
    -BaseUrl $BaseUrl
