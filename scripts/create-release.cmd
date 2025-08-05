@echo off
setlocal enabledelayedexpansion

echo ========================================
echo PlexiChat Client - GitHub Release Creator
echo ========================================
echo.

REM Set version (can be overridden with environment variable)
if "%RELEASE_VERSION%"=="" set RELEASE_VERSION=v1.0.0

echo Creating GitHub release for %RELEASE_VERSION%
echo.

REM Check if release directory exists
set RELEASE_DIR=releases\%RELEASE_VERSION%
if not exist "%RELEASE_DIR%" (
    echo ❌ Release directory not found: %RELEASE_DIR%
    echo Please run build-release.cmd first
    pause
    exit /b 1
)

echo [1/3] Creating GitHub release...

REM Create release notes
set RELEASE_NOTES=PlexiChat Client %RELEASE_VERSION% - Production Release

🚀 **Complete CLI and GUI applications for PlexiChat messaging platform**

## ✨ Features
- **Modern messaging interface** with real-time communication
- **Full-featured CLI** with interactive commands and configuration
- **ASCII-only logging system** with configurable levels and colorization
- **Comprehensive configuration management** (YAML, env vars, CLI flags)
- **Advanced security validation** and XSS protection
- **WebSocket real-time messaging** with auto-reconnect
- **File upload** with drag & drop support and emoji picker
- **Advanced retry logic** with exponential backoff
- **Auto-update functionality** for seamless updates
- **Professional documentation** and deployment guides

## 🛠️ Installation
Download the appropriate package for your platform:
- **Windows**: `plexichat-client-%RELEASE_VERSION%-windows-amd64.zip`
- **Linux**: `plexichat-client-%RELEASE_VERSION%-linux-amd64.tar.gz`
- **macOS**: `plexichat-client-%RELEASE_VERSION%-macos-amd64.tar.gz`

## 🚀 Quick Start
```bash
# Extract the package
# Windows: Extract the .zip file
# Linux/macOS: tar -xzf plexichat-client-*.tar.gz

# Initialize configuration
./plexichat-cli config init

# Set server URL
./plexichat-cli config set url "http://localhost:8000"

# Start CLI
./plexichat-cli chat

# Or launch GUI
./plexichat-gui
```

## 📚 Documentation
- Complete README with feature overview
- Configuration guide (docs/CONFIGURATION.md)
- Troubleshooting guide (docs/TROUBLESHOOTING.md)
- API documentation (docs/API.md)
- Deployment guide (DEPLOYMENT_GUIDE.md)

## 🎯 Technical Details
- **Go 1.19+** with Fyne GUI framework
- **Cross-platform** support (Windows, macOS, Linux)
- **Modular architecture** with comprehensive testing
- **Professional error handling** and recovery
- **Production ready** with deployment instructions

Ready for production use! 🎉

REM Create the release
gh release create "%RELEASE_VERSION%" ^
    --title "PlexiChat Client %RELEASE_VERSION%" ^
    --notes "%RELEASE_NOTES%" ^
    --latest

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to create GitHub release
    pause
    exit /b 1
)

echo [2/3] Uploading release assets...

REM Upload Windows package
echo   Uploading Windows package...
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-windows-amd64.zip"

REM Upload Linux package
echo   Uploading Linux package...
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-linux-amd64.tar.gz"

REM Upload macOS package
echo   Uploading macOS package...
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-macos-amd64.tar.gz"

REM Upload individual binaries for direct download
echo   Uploading individual binaries...
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-cli-windows-amd64.exe"
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-gui-windows-amd64.exe"
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-cli-linux-amd64"
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-gui-linux-amd64"
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-cli-macos-amd64"
gh release upload "%RELEASE_VERSION%" "%RELEASE_DIR%\plexichat-gui-macos-amd64"

echo [3/3] Finalizing release...

REM Show release info
gh release view "%RELEASE_VERSION%"

echo.
echo ========================================
echo 🎉 GitHub Release Created Successfully!
echo ========================================
echo.
echo Release: %RELEASE_VERSION%
echo URL: https://github.com/linux-of-user/plexichat-app/releases/tag/%RELEASE_VERSION%
echo.
echo Assets uploaded:
echo ✅ Windows package (ZIP)
echo ✅ Linux package (TAR.GZ)
echo ✅ macOS package (TAR.GZ)
echo ✅ Individual binaries for all platforms
echo.
echo 🚀 Release is now live and ready for download!
echo.
pause
