param(
    [string]$InstallDir = "$HOME\.local\bin",
    [string]$Version = "",
    [string]$Repo = "vastimofeev/yandex-tracker-cli",
    [switch]$FromRelease,
    [switch]$SkipPathUpdate,
    [ValidateSet("local", "github")]
    [string]$Channel = ""
)

$ErrorActionPreference = 'Stop'

$installArgs = @{
    InstallDir      = $InstallDir
    Version         = $Version
    Repo            = $Repo
    FromRelease     = $FromRelease
    SkipPathUpdate  = $SkipPathUpdate
}

if ($Channel) {
    $installArgs.Channel = $Channel
}

& (Join-Path $PSScriptRoot "install.ps1") @installArgs
