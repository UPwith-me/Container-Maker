# Container Maker Development Script
# Starts both frontend (watch/build) and backend

Write-Host "🚀 Starting Container Maker Development Environment..." -ForegroundColor Green

# Check for Node.js
if (!(Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Error "Node.js is required for development/building frontend. Please install it."
    exit 1
}

# Check for Go
if (!(Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go is required for backend. Please install it."
    exit 1
}

# Build Frontend (One-time build for embed, but we can also run dev server if needed)
Write-Host "📦 Building Frontend..." -ForegroundColor Cyan
Set-Location "cloud\ui"
npm install
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Error "Frontend build failed"
    exit 1
}
Set-Location "..\.."

# Run Backend
Write-Host "🔥 Starting Backend (with embedded frontend)..." -ForegroundColor Green
Write-Host "👉 Open http://localhost:8080 in your browser" -ForegroundColor Yellow

# Set CGO_ENABLED=0 to ensure pure Go build
$env:CGO_ENABLED = "0"
go run ./cmd/server
