# GIOS Windows Native Installer
# This script downloads and installs the gios binary to $HOME\.gios\bin

$Repo = "nikitacontreras/gios"
$GiosDir = Join-Path $HOME ".gios"
$BinDir = Join-Path $GiosDir "bin"

Write-Host ""
Write-Host " dP`"`"b8 88  dP`"Yb  .dP`"Y8 " -ForegroundColor Cyan
Write-Host "dP   `"`" 88 dP   Yb ``Ybo.`" " -ForegroundColor Cyan
Write-Host "Yb  `"88 88 Yb   dP o.``Y8b " -ForegroundColor Cyan
Write-Host " YboodP 88  YbodP  8bodP' " -ForegroundColor Cyan
Write-Host ""
Write-Host "--------------------------------------------------" -ForegroundColor Cyan

# Detect Architecture
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

# Ensure bin directory exists
if (!(Test-Path $BinDir)) {
    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
}

Write-Host "[1/3] Fetching latest release info..." -ForegroundColor White
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -Method Get
    $Tag = $Release.tag_name
} catch {
    Write-Host " [!] Stable release not found, falling back to nightly..." -ForegroundColor Yellow
    $Tag = "latest"
}

$BinaryName = "gios-windows-$Arch.exe"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$BinaryName"
$DestPath = Join-Path $BinDir "gios.exe"

Write-Host "[2/3] Downloading GIOS $Tag ($Arch)..." -ForegroundColor White
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $DestPath
} catch {
    Write-Host " [!] Error downloading binary: $_" -ForegroundColor Red
    exit
}

Write-Host "[3/3] Finalizing installation..." -ForegroundColor White

# Update PATH if needed
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -notlike "*$BinDir*") {
    Write-Host " [+] Adding GIOS to User PATH..." -ForegroundColor Green
    $NewPath = "$CurrentPath;$BinDir"
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    $env:Path = "$env:Path;$BinDir"
}

Write-Host "--------------------------------------------------" -ForegroundColor Cyan
Write-Host "✅ GIOS installed successfully to $BinDir\gios.exe" -ForegroundColor Green
Write-Host "   Please restart your terminal to use 'gios'." -ForegroundColor White
Write-Host "   Try running: gios doctor" -ForegroundColor White
Write-Host "--------------------------------------------------" -ForegroundColor Cyan
