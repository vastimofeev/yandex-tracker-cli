param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$DistDir = ".\dist",
    [string]$Bucket = "yandex-tracker-cli",
    [string]$Endpoint = "https://s3.ru-1.storage.selcloud.ru",
    [string]$Region = "ru-1"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
    throw "aws CLI is required to publish release assets to Selectel S3"
}

$resolvedDist = Resolve-Path $DistDir
$checksums = Join-Path $resolvedDist "checksums.txt"
$assets = @(
    "yt-windows-amd64.exe",
    "yt-linux-amd64",
    "yt-darwin-amd64",
    "yt-darwin-arm64"
)

$hashLines = @()
foreach ($asset in $assets) {
    $path = Join-Path $resolvedDist $asset
    if (-not (Test-Path $path)) {
        throw "missing release asset: $path"
    }
    $hash = (Get-FileHash -Algorithm SHA256 -Path $path).Hash.ToLowerInvariant()
    $hashLines += "$hash  $asset"
}
Set-Content -Path $checksums -Value ($hashLines -join [Environment]::NewLine)

$s3ReleasePrefix = "s3://$Bucket/releases/$Version"
$s3LatestPrefix = "s3://$Bucket/latest"

foreach ($asset in $assets) {
    $path = Join-Path $resolvedDist $asset
    aws --endpoint-url $Endpoint --region $Region s3 cp $path "$s3ReleasePrefix/$asset"
}

aws --endpoint-url $Endpoint --region $Region s3 cp $checksums "$s3ReleasePrefix/checksums.txt"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $resolvedDist "yt-windows-amd64.exe") "$s3LatestPrefix/windows/yt.exe"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $resolvedDist "yt-linux-amd64") "$s3LatestPrefix/yt-linux-amd64"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $resolvedDist "yt-darwin-amd64") "$s3LatestPrefix/yt-darwin-amd64"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $resolvedDist "yt-darwin-arm64") "$s3LatestPrefix/yt-darwin-arm64"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $PSScriptRoot "install.ps1") "$s3LatestPrefix/install.ps1"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $PSScriptRoot "update.ps1") "$s3LatestPrefix/update.ps1"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $PSScriptRoot "..\install-yt.bat") "$s3LatestPrefix/install-yt.bat"
aws --endpoint-url $Endpoint --region $Region s3 cp (Join-Path $PSScriptRoot "..\update-yt.bat") "$s3LatestPrefix/update-yt.bat"

Write-Host "Uploaded release $Version to $Endpoint/$Bucket"
