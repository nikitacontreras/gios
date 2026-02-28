$ErrorActionPreference = "Stop"

$GiosBinDir = Join-Path $env:USERPROFILE ".gios\bin"

Write-Host "=> Compiling gios CLI..."
go build -o gios.exe main.go

Write-Host "=> Creating installation directory at $GiosBinDir..."
if (-not (Test-Path -Path $GiosBinDir)) {
    New-Item -ItemType Directory -Path $GiosBinDir | Out-Null
}

Write-Host "=> Installing gios to $GiosBinDir..."
Copy-Item -Path "gios.exe" -Destination (Join-Path $GiosBinDir "gios.exe") -Force
Remove-Item -Path "gios.exe" -Force

Write-Host "=> Mission accomplished!"

$PathMatch = ($env:PATH -split ';') -contains $GiosBinDir
if (-not $PathMatch) {
    Write-Host "NOTE: The installation directory is not in your PATH." -ForegroundColor Yellow
    Write-Host "You can add it by running the following command:"
    Write-Host "[Environment]::SetEnvironmentVariable(`"Path`", `$env:Path + `";$GiosBinDir`", `"User`")"
} else {
    Write-Host "The 'gios' tool has been successfully installed or updated!" -ForegroundColor Green
}
