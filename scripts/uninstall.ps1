param(
    [string]$InstallDir = "$HOME\.local\bin",
    [switch]$SkipPathUpdate
)

$ErrorActionPreference = 'Stop'

$target = Join-Path $InstallDir "yt.exe"

if (Test-Path $target) {
    Remove-Item -LiteralPath $target -Force
    Write-Host "Removed $target"
} else {
    Write-Host "yt.exe is not installed in $InstallDir"
}

if (-not $SkipPathUpdate) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath) {
        $parts = $userPath.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries) |
            Where-Object { $_ -ne $InstallDir }
        $newPath = ($parts | Select-Object -Unique) -join ';'
        try {
            [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
            Write-Host "Removed $InstallDir from user PATH if it was present."
        } catch {
            Write-Warning "Could not update user PATH automatically: $($_.Exception.Message)"
        }
    }
}

Write-Host "Uninstall completed."
