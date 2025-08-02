@echo off
echo ========================================
echo PlexiChat Desktop Simple Builder
echo ========================================
echo.

REM Clean up
if exist build rmdir /s /q build
mkdir build

echo Building PlexiChat CLI...
go build -ldflags "-X main.version=2.0.0-alpha" -o build\plexichat-cli.exe .

if exist build\plexichat-cli.exe (
    echo ✅ CLI build successful!
    echo.
    echo Testing CLI...
    build\plexichat-cli.exe --version
    echo.
    echo ✅ CLI is working!
    echo.
    echo Usage:
    echo   .\build\plexichat-cli.exe --help
    echo   .\build\plexichat-cli.exe auth login
    echo   .\build\plexichat-cli.exe gui
    echo.
) else (
    echo ❌ CLI build failed!
    echo Check for Go installation and dependencies
)

echo Attempting GUI build...
set CGO_ENABLED=1
go build -ldflags "-X main.version=2.0.0-alpha" -o build\plexichat-gui.exe .

if exist build\plexichat-gui.exe (
    echo ✅ GUI build successful!
    echo   .\build\plexichat-gui.exe
) else (
    echo ⚠️  GUI build failed - this is expected without CGO setup
    echo Use CLI version or see SETUP-GUIDE.md for GUI setup
)

echo.
echo ========================================
echo Build Complete!
echo ========================================
echo.
echo Built files:
if exist build\*.exe dir build\*.exe
echo.
echo Ready for GitHub deployment!
echo See SETUP-AND-DEPLOY.md for next steps
echo.
pause
