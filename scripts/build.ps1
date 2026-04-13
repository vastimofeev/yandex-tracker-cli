param(
    [string]$Output = ".\yandex-tracker-cli.exe",
    [string]$Version = "",
    [string]$Commit = "",
    [string]$BuildDate = ""
)

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot

if (-not $Version) {
    try {
        $Version = (git -C $repoRoot describe --tags --always 2>$null).Trim()
    } catch {
        $Version = "dev"
    }
    if (-not $Version) { $Version = "dev" }
}

if (-not $Commit) {
    try {
        $Commit = (git -C $repoRoot rev-parse --short HEAD 2>$null).Trim()
    } catch {
        $Commit = "unknown"
    }
    if (-not $Commit) { $Commit = "unknown" }
}

if (-not $BuildDate) {
    $BuildDate = Get-Date -Format "yyyy-MM-ddTHH:mm:ssK"
}

$ldflags = "-X main.buildVersion=$Version -X main.buildCommit=$Commit -X main.buildDate=$BuildDate"

Push-Location $repoRoot
try {
    go build -ldflags $ldflags -o $Output ./cmd/yandex-tracker-cli
} finally {
    Pop-Location
}
