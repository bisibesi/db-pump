# build.ps1 - DB Pump Release Build Script

$DistDir = "dist"

# 1. Cleanup
Write-Host "🧹 Cleaning up $DistDir..." -ForegroundColor Yellow
if (Test-Path $DistDir) {
    Remove-Item $DistDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

# 2. Build Windows
Write-Host "🪟 Building for Windows (amd64)..." -ForegroundColor Cyan
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags "-s -w" -o "$DistDir/db-pump.exe" .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Windows build failed!"
    exit 1
}

# Copy Config File
Write-Host "📂 Copying configuration..." -ForegroundColor Cyan
Copy-Item "db-pump.yaml" -Destination "$DistDir/db-pump.yaml"

# 3. Build Linux
Write-Host "🐧 Building for Linux (amd64)..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags "-s -w" -o "$DistDir/db-pump-linux" .
if ($LASTEXITCODE -ne 0) {
    Write-Error "Linux build failed!"
    exit 1
}

# Reset Environment Variables (Optional, ensures local dev stays on defaults)
$env:GOOS = ""
$env:GOARCH = ""

# 4. Completion
Write-Host "🦅 Build Complete! Check ./$DistDir folder" -ForegroundColor Green
Get-ChildItem $DistDir | Select-Object Name, @{Name = "Size(KB)"; Expression = { [math]::round($_.Length / 1KB, 2) } }
