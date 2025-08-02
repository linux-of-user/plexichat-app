#!/usr/bin/env pwsh

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "PlexiChat Desktop Working Builder" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Clean up
Write-Host "🧹 Cleaning previous builds..." -ForegroundColor Yellow
if (Test-Path "build") {
    Remove-Item -Recurse -Force "build"
}
New-Item -ItemType Directory -Path "build" | Out-Null

# Test Go
Write-Host "🔍 Testing Go environment..." -ForegroundColor White
try {
    $goVersion = & go version
    Write-Host "✅ Go is available: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go is not available" -ForegroundColor Red
    exit 1
}

# Test basic CLI build
Write-Host ""
Write-Host "🏗️ Building basic CLI..." -ForegroundColor White
try {
    & go build -o "build/plexichat-cli.exe" .
    if (Test-Path "build/plexichat-cli.exe") {
        Write-Host "✅ CLI build successful" -ForegroundColor Green
    } else {
        throw "CLI executable not created"
    }
} catch {
    Write-Host "❌ CLI build failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test CLI execution
Write-Host ""
Write-Host "🧪 Testing CLI execution..." -ForegroundColor White
try {
    $output = & "build/plexichat-cli.exe" --version 2>&1
    Write-Host "✅ CLI execution successful: $output" -ForegroundColor Green
} catch {
    Write-Host "❌ CLI execution failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test simple CLI
Write-Host ""
Write-Host "🧪 Building simple test CLI..." -ForegroundColor White
try {
    & go build -o "build/test-cli.exe" test-cli.go
    if (Test-Path "build/test-cli.exe") {
        Write-Host "✅ Test CLI build successful" -ForegroundColor Green
        
        # Test execution
        $testOutput = & "build/test-cli.exe" test 2>&1
        Write-Host "Test output: $testOutput" -ForegroundColor Cyan
    }
} catch {
    Write-Host "⚠️ Test CLI build failed: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Check CGO status
Write-Host ""
Write-Host "🔍 Checking CGO status..." -ForegroundColor White
$cgoStatus = & go env CGO_ENABLED
Write-Host "CGO_ENABLED: $cgoStatus" -ForegroundColor Cyan

if ($cgoStatus -eq "0") {
    Write-Host "⚠️ CGO is disabled - GUI build will fail" -ForegroundColor Yellow
    Write-Host "Setting CGO_ENABLED=1..." -ForegroundColor White
    $env:CGO_ENABLED = "1"
}

# Check for C compiler
Write-Host ""
Write-Host "🔍 Checking for C compiler..." -ForegroundColor White
try {
    $gccVersion = & gcc --version 2>&1
    Write-Host "✅ GCC found: $($gccVersion[0])" -ForegroundColor Green
    $hasCompiler = $true
} catch {
    try {
        $clVersion = & cl 2>&1
        Write-Host "✅ MSVC found" -ForegroundColor Green
        $hasCompiler = $true
    } catch {
        Write-Host "⚠️ No C compiler found - GUI build will fail" -ForegroundColor Yellow
        $hasCompiler = $false
    }
}

# Try GUI build if compiler available
if ($hasCompiler) {
    Write-Host ""
    Write-Host "🎨 Attempting GUI build..." -ForegroundColor White
    try {
        $env:CGO_ENABLED = "1"
        & go build -tags gui -o "build/plexichat-gui.exe" .
        if (Test-Path "build/plexichat-gui.exe") {
            Write-Host "✅ GUI build successful!" -ForegroundColor Green
            
            # Test GUI execution (just version check)
            try {
                $guiOutput = & "build/plexichat-gui.exe" --version 2>&1
                Write-Host "✅ GUI execution test: $guiOutput" -ForegroundColor Green
            } catch {
                Write-Host "⚠️ GUI execution test failed: $($_.Exception.Message)" -ForegroundColor Yellow
            }
        } else {
            Write-Host "⚠️ GUI executable not created" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "⚠️ GUI build failed: $($_.Exception.Message)" -ForegroundColor Yellow
        Write-Host "This is expected without proper CGO setup" -ForegroundColor Cyan
    }
} else {
    Write-Host "⚠️ Skipping GUI build - no C compiler" -ForegroundColor Yellow
}

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Build Summary" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green

Write-Host ""
Write-Host "Built files:" -ForegroundColor White
Get-ChildItem "build/*.exe" -ErrorAction SilentlyContinue | ForEach-Object {
    $size = [math]::Round($_.Length / 1KB, 2)
    Write-Host "  ✅ $($_.Name) ($size KB)" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "Usage:" -ForegroundColor White
if (Test-Path "build/plexichat-cli.exe") {
    Write-Host "  CLI: .\build\plexichat-cli.exe --help" -ForegroundColor Cyan
}
if (Test-Path "build/plexichat-gui.exe") {
    Write-Host "  GUI: .\build\plexichat-gui.exe" -ForegroundColor Cyan
}
if (Test-Path "build/test-cli.exe") {
    Write-Host "  Test: .\build\test-cli.exe test" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "🎉 Build process complete!" -ForegroundColor Green

# Test the CLI help
if (Test-Path "build/plexichat-cli.exe") {
    Write-Host ""
    Write-Host "📋 Testing CLI help..." -ForegroundColor White
    try {
        & "build/plexichat-cli.exe" --help
    } catch {
        Write-Host "CLI help test failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "Next Steps:" -ForegroundColor Green
Write-Host "1. Test: .\build\plexichat-cli.exe --version" -ForegroundColor Cyan
Write-Host "2. GUI: .\build\plexichat-gui.exe (if available)" -ForegroundColor Cyan
Write-Host "3. Help: .\build\plexichat-cli.exe --help" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Green
