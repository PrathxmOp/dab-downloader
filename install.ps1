# DAB Downloader Windows Installer
# Usage (PowerShell): iwr -useb https://raw.githubusercontent.com/PrathxmOp/dab-downloader/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$repo = "PrathxmOp/dab-downloader"
$binaryName = "dab-downloader.exe"
$installDir = "$HOME\.dab-downloader"

Write-Host "🎵 DAB Downloader Windows Installer" -ForegroundColor Blue

# Detect Architecture
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") { "amd64" } else { "386" }
Write-Host "🔍 Detected Architecture: $arch"

# Fetch latest release info from GitHub API
Write-Host "📡 Fetching latest release info..."
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name

Write-Host "✨ Latest version: $version"

# Construct download URL
$assetName = "dab-downloader-windows-$arch.exe"
$asset = $release.assets | Where-Object { $_.name -eq $assetName }

if (-not $asset) {
    Write-Host "❌ Could not find a download URL for $assetName." -ForegroundColor Red
    exit 1
}

$downloadUrl = $asset.browser_download_url

# Create installation directory
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

# Download
Write-Host "📥 Downloading $assetName..."
Invoke-WebRequest -Uri $downloadUrl -OutFile "$installDir\$binaryName"

# Add to PATH if not already there
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$installDir*") {
    Write-Host "🚀 Adding $installDir to User PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
    $env:Path += ";$installDir"
    Write-Host "✅ PATH updated. You may need to restart your terminal." -ForegroundColor Cyan
}

Write-Host "✅ Successfully installed DAB Downloader $version!" -ForegroundColor Green
Write-Host "Run 'dab-downloader version' to verify."
