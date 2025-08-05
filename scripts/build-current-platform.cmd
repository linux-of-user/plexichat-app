@echo off
setlocal enabledelayedexpansion

echo ========================================
echo PlexiChat Client - Current Platform Builder
echo ========================================
echo.

REM Set version (can be overridden with environment variable)
if "%RELEASE_VERSION%"=="" set RELEASE_VERSION=v1.0.0

echo Building PlexiChat Client %RELEASE_VERSION% for current platform
echo.

REM Create release directory
set RELEASE_DIR=releases\%RELEASE_VERSION%
if exist "%RELEASE_DIR%" rmdir /s /q "%RELEASE_DIR%"
mkdir "%RELEASE_DIR%"

echo [1/3] Building CLI application...
set CGO_ENABLED=1
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%\plexichat-cli.exe" plexichat-cli.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build CLI
    exit /b 1
)
echo ✅ CLI built successfully

echo [2/3] Building GUI application...
go build -ldflags "-X plexichat-client/pkg/updater.CurrentVersion=%RELEASE_VERSION%" -o "%RELEASE_DIR%\plexichat-gui.exe" plexichat-gui.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to build GUI
    exit /b 1
)
echo ✅ GUI built successfully

echo [3/3] Creating package...
powershell -Command "Compress-Archive -Path '%RELEASE_DIR%\*.exe' -DestinationPath '%RELEASE_DIR%\plexichat-client-%RELEASE_VERSION%-windows-amd64.zip'"
echo ✅ Package created

echo.
echo ========================================
echo ✅ Build Complete!
echo ========================================
echo.
echo Release files created in: %RELEASE_DIR%
echo.
echo Files:
dir /b "%RELEASE_DIR%"
echo.
echo Ready to test!
echo.
pause
