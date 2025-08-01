@echo off
setlocal enabledelayedexpansion

echo ========================================
echo PlexiChat Release Builder v2.0.0-alpha
echo ========================================
echo.

REM Clean up old builds
echo Cleaning previous builds...
if exist "build" rmdir /s /q "build"
if exist "release" rmdir /s /q "release"
mkdir build
mkdir release

REM Set version info
set VERSION=2.0.0-alpha
set BUILD_TIME=%date% %time%
set COMMIT=HEAD

echo Building PlexiChat v%VERSION%...
echo Build Time: %BUILD_TIME%
echo.

REM Set common build flags
set LDFLAGS=-ldflags "-X main.version=%VERSION% -X main.buildTime=%BUILD_TIME% -X main.commit=%COMMIT%"

REM Build Windows GUI (Fyne)
echo ========================================
echo Building Windows GUI Application...
echo ========================================
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

fyne package -name PlexiChat-GUI -icon icon.png
if exist "PlexiChat-GUI.exe" (
    move "PlexiChat-GUI.exe" "build\PlexiChat-GUI-windows-amd64.exe"
    echo ✓ Windows GUI build successful
) else (
    echo ✗ Windows GUI build failed
)

REM Build Windows CLI
echo.
echo Building Windows CLI Application...
go build %LDFLAGS% -o build\PlexiChat-CLI-windows-amd64.exe .
if exist "build\PlexiChat-CLI-windows-amd64.exe" (
    echo ✓ Windows CLI build successful
) else (
    echo ✗ Windows CLI build failed
)

REM Build Linux GUI (if possible)
echo.
echo ========================================
echo Building Linux Applications...
echo ========================================
set GOOS=linux
set GOARCH=amd64

echo Building Linux CLI...
go build %LDFLAGS% -o build\PlexiChat-CLI-linux-amd64 .
if exist "build\PlexiChat-CLI-linux-amd64" (
    echo ✓ Linux CLI build successful
) else (
    echo ✗ Linux CLI build failed
)

REM Build macOS CLI
echo.
echo ========================================
echo Building macOS Applications...
echo ========================================
set GOOS=darwin
set GOARCH=amd64

echo Building macOS CLI (Intel)...
go build %LDFLAGS% -o build\PlexiChat-CLI-darwin-amd64 .
if exist "build\PlexiChat-CLI-darwin-amd64" (
    echo ✓ macOS Intel CLI build successful
) else (
    echo ✗ macOS Intel CLI build failed
)

set GOARCH=arm64
echo Building macOS CLI (Apple Silicon)...
go build %LDFLAGS% -o build\PlexiChat-CLI-darwin-arm64 .
if exist "build\PlexiChat-CLI-darwin-arm64" (
    echo ✓ macOS Apple Silicon CLI build successful
) else (
    echo ✗ macOS Apple Silicon CLI build failed
)

echo.
echo ========================================
echo Creating Release Packages...
echo ========================================

REM Create Windows release
if exist "build\PlexiChat-GUI-windows-amd64.exe" (
    echo Creating Windows release package...
    mkdir "release\windows"
    copy "build\PlexiChat-GUI-windows-amd64.exe" "release\windows\"
    copy "build\PlexiChat-CLI-windows-amd64.exe" "release\windows\"
    copy "README.md" "release\windows\"
    copy "icon.png" "release\windows\"
    
    cd release
    powershell Compress-Archive -Path "windows\*" -DestinationPath "PlexiChat-v%VERSION%-windows-amd64.zip" -Force
    cd ..
    echo ✓ Windows package created
)

echo.
echo ========================================
echo Build Summary
echo ========================================
echo Version: %VERSION%
echo Build Time: %BUILD_TIME%
echo.
echo Built applications:
dir build\PlexiChat-* /b 2>nul
echo.
echo Release packages:
dir release\*.zip /b 2>nul
echo.
echo ========================================
echo Build Complete!
echo ========================================
pause
