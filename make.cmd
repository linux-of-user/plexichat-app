@echo off
echo ================================================================================
echo PlexiChat Desktop Builder
echo ================================================================================
echo.

REM Clean up old builds
if exist build rmdir /s /q build
mkdir build

echo Step 1: Building CLI version...
go build -ldflags "-X main.version=2.0.0-alpha" -o build\plexichat.exe .
if errorlevel 1 (
    echo [ERROR] CLI build failed
    pause
    exit /b 1
)
echo [OK] CLI build successful: build\plexichat.exe

echo.
echo Step 2: Testing CLI...
build\plexichat.exe --version
if errorlevel 1 (
    echo [ERROR] CLI execution failed
    pause
    exit /b 1
)
echo [OK] CLI is working

echo.
echo Step 3: Building GUI version (requires CGO)...
set CGO_ENABLED=1
go build -ldflags "-X main.version=2.0.0-alpha -H windowsgui" -o build\plexichat-gui.exe gui-main.go
if errorlevel 1 (
    echo [WARNING] GUI build failed - CGO/C compiler may not be available
    echo [INFO] CLI version is ready in build\plexichat.exe
) else (
    echo [OK] GUI build successful: build\plexichat-gui.exe (no terminal window)
)

echo Step 4: Building GUI debug version...
go build -ldflags "-X main.version=2.0.0-alpha" -o build\plexichat-gui-debug.exe gui-main.go
if errorlevel 1 (
    echo [WARNING] GUI debug build failed
) else (
    echo [OK] GUI debug build successful: build\plexichat-gui-debug.exe (with terminal)
)

echo.
echo ================================================================================
echo Build Complete!
echo ================================================================================
echo.
echo Built files:
if exist build\*.exe dir build\*.exe
echo.
echo Usage:
echo   build\plexichat.exe --help
echo   build\plexichat.exe demo
echo   build\plexichat.exe gui --debug (CLI with GUI debug output)
echo   build\plexichat-gui.exe (production GUI - no terminal)
echo   build\plexichat-gui-debug.exe --debug (GUI with debug terminal)
echo.
pause
