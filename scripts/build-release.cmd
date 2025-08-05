@echo off
setlocal enabledelayedexpansion

echo ========================================
echo PlexiChat Client - Release Builder
echo ========================================
echo.

REM Set version (can be overridden with environment variable)
if "%RELEASE_VERSION%"=="" set RELEASE_VERSION=v1.0.0

echo Building PlexiChat Client %RELEASE_VERSION%
echo.

REM Create release directory
set RELEASE_DIR=releases\%RELEASE_VERSION%
if exist "%RELEASE_DIR%" rmdir /s /q "%RELEASE_DIR%"
mkdir "%RELEASE_DIR%"

echo [1/4] Building Windows binaries...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM Build CLI
echo   Building CLI for Windows...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%\plexichat-cli-windows-amd64.exe" plexichat-cli.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build Windows CLI
    exit /b 1
)

REM Build GUI
echo   Building GUI for Windows...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%\plexichat-gui-windows-amd64.exe" plexichat-gui.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build Windows GUI
    exit /b 1
)

echo [2/4] Building Linux binaries...
set GOOS=linux
set GOARCH=amd64

REM Build CLI for Linux
echo   Building CLI for Linux...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%/plexichat-cli-linux-amd64" plexichat-cli.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build Linux CLI
    exit /b 1
)

REM Build GUI for Linux
echo   Building GUI for Linux...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%/plexichat-gui-linux-amd64" plexichat-gui.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build Linux GUI
    exit /b 1
)

echo [3/4] Building macOS binaries...
set GOOS=darwin
set GOARCH=amd64

REM Build CLI for macOS
echo   Building CLI for macOS...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%/plexichat-cli-macos-amd64" plexichat-cli.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build macOS CLI
    exit /b 1
)

REM Build GUI for macOS
echo   Building GUI for macOS...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%/plexichat-gui-macos-amd64" plexichat-gui.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build macOS GUI
    exit /b 1
)

echo [4/4] Creating release packages...

REM Create Windows package
echo   Creating Windows package...
powershell -Command "Compress-Archive -Path '%RELEASE_DIR%\*windows*' -DestinationPath '%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-windows-amd64.zip'"

REM Create Linux package
echo   Creating Linux package...
tar -czf "%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-linux-amd64.tar.gz" -C "%RELEASE_DIR%" plexichat-cli-linux-amd64 plexichat-gui-linux-amd64

REM Create macOS package
echo   Creating macOS package...
tar -czf "%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-macos-amd64.tar.gz" -C "%RELEASE_DIR%" plexichat-cli-macos-amd64 plexichat-gui-macos-amd64

echo.
echo ========================================
echo ✅ Build Complete!
echo ========================================
echo.
echo Release files created in: %RELEASE_DIR%
echo.
echo Files:
dir /b "%RELEASE_DIR%\*.zip" "%RELEASE_DIR%\*.tar.gz" 2>nul
echo.
echo Individual binaries:
dir /b "%RELEASE_DIR%\plexichat-*" 2>nul
echo.
echo Ready to create GitHub release!
echo.
pause
