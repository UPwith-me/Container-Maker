# Container-Maker Release Packager
# Builds cross-platform binaries and packages them for release

$ErrorActionPreference = "Stop"
$Version = "v2.0.0"

Write-Host "📦 Starting Container-Maker $Version Packaging..." -ForegroundColor Green
Write-Host "==============================================" -ForegroundColor Green

# 1. Verification
Write-Host "`n🔍 Running Verification..." -ForegroundColor Cyan
Write-Host "   > Running go vet..."
go vet ./...
if ($LASTEXITCODE -ne 0) { Write-Error "go vet failed"; exit 1 }

Write-Host "   > Running go test..."
go test ./...
if ($LASTEXITCODE -ne 0) { Write-Error "go test failed"; exit 1 }
Write-Host "✅ Verification Passed" -ForegroundColor Green

# 2. Prepare Output Directory
$DistDir = "dist"
if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
New-Item -ItemType Directory -Path $DistDir | Out-Null
Write-Host "`n📁 Created $DistDir directory" -ForegroundColor Cyan

# 3. Build Binaries
$Builds = @(
    @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" },
    @{ OS = "linux"; Arch = "amd64"; Ext = "" },
    @{ OS = "darwin"; Arch = "amd64"; Ext = "" },
    @{ OS = "darwin"; Arch = "arm64"; Ext = "" }
)

foreach ($Build in $Builds) {
    $Target = "cm-$($Build.OS)-$($Build.Arch)$($Build.Ext)"
    $OutPath = Join-Path $DistDir $Target
    
    Write-Host "🔨 Building for $($Build.OS)/$($Build.Arch)..." -ForegroundColor Yellow
    
    $env:GOOS = $Build.OS
    $env:GOARCH = $Build.Arch
    $env:CGO_ENABLED = "0"
    
    go build -ldflags="-s -w -X main.Version=$Version" -o $OutPath ./cmd/cm
    
    if ($LASTEXITCODE -ne 0) { 
        Write-Error "Build failed for $($Build.OS)/$($Build.Arch)"
        exit 1 
    }
}

# 4. Package Assets
Write-Host "`n📄 Copying documentation..." -ForegroundColor Cyan
Copy-Item "README.md" $DistDir
Copy-Item "README_CN.md" $DistDir -ErrorAction SilentlyContinue
Copy-Item "LICENSE" $DistDir -ErrorAction SilentlyContinue

# 5. Build VS Code Extension (Optional)
if (Test-Path "extensions/vscode") {
    Write-Host "`n🧩 Packaging VS Code Extension..." -ForegroundColor Cyan
    Push-Location "extensions/vscode"
    try {
        if (Get-Command npm -ErrorAction SilentlyContinue) {
            Write-Host "   > installing dependencies..."
            npm install --silent
            Write-Host "   > compiling..."
            npm run compile
            
            # Simple zip of the extension folder for now as we don't assume vsce is installed
            $ExtDist = Join-Path "..\..\" $DistDir "vscode-extension"
            New-Item -ItemType Directory -Path $ExtDist -Force | Out-Null
            Copy-Item "*" $ExtDist -Recurse -Exclude "node_modules", ".git"
            Write-Host "   ✅ Extension source packaged" -ForegroundColor Green
        }
        else {
            Write-Host "   ⚠️ npm not found, skipping extension build" -ForegroundColor Yellow
        }
    }
    catch {
        Write-Host "   ⚠️ Extension packaging failed: $_" -ForegroundColor Red
    }
    Pop-Location
}

# 6. Create Final Archive (Zip)
Write-Host "`n📦 Creating final archive..." -ForegroundColor Cyan
$ZipPath = "Container-Maker-$Version.zip"
Compress-Archive -Path "$DistDir\*" -DestinationPath $ZipPath -Force

Write-Host "`n==============================================" -ForegroundColor Green
Write-Host "✅ Release Ready: $ZipPath" -ForegroundColor Green
Write-Host "📁 Artifacts in: $DistDir" -ForegroundColor Green
