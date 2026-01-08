# Container Maker Release Builder
# Builds the single binary with embedded frontend

$ErrorActionPreference = "Stop"
Write-Host "🏗️  Building Container Maker Release..." -ForegroundColor Green

# 1. Build Frontend
Write-Host "1️⃣  Building Frontend..." -ForegroundColor Cyan
Set-Location "cloud\ui"
npm install --silent
npm run build
if ($LASTEXITCODE -ne 0) { Write-Error "Frontend build failed"; exit 1 }
Set-Location "..\.."

# 2. Build Backend (Pure Go, Optimized)
Write-Host "2️⃣  Compiling Go Binary (Static/Pure Go/Optimized)..." -ForegroundColor Cyan
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o cm-control-plane.exe ./cmd/server

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Build Success!" -ForegroundColor Green
    Write-Host "👉 Binary: .\cm-control-plane.exe" -ForegroundColor Yellow
    Write-Host "👉 Usage: Just double-click it. No dependencies needed." -ForegroundColor Gray
}
else {
    Write-Error "Backend build failed"
}
