@echo off
echo Cleaning up PlexiChat build environment...

REM Remove old build artifacts
if exist "PlexiChat.exe" del /f "PlexiChat.exe"
if exist "build" rmdir /s /q "build"
if exist "release" rmdir /s /q "release"
if exist "docs-test" rmdir /s /q "docs-test"
if exist "test-docs" rmdir /s /q "test-docs"
if exist "build-gui.bat" del /f "build-gui.bat"
if exist "build-all-platforms.bat" del /f "build-all-platforms.bat"
if exist "tdm-gcc-installer.exe" del /f "tdm-gcc-installer.exe"
if exist "tdm-gcc.html" del /f "tdm-gcc.html"
if exist "mingw.7z" del /f "mingw.7z"

REM Create clean build directory
mkdir build

echo Cleanup complete!
echo.
echo Building PlexiChat GUI...

REM Set environment for CGO
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM Build the GUI application
echo Building Windows GUI application...
fyne package -name PlexiChat -icon icon.png -o build\PlexiChat-GUI.exe

if errorlevel 1 (
    echo GUI build failed, trying fallback build...
    go build -tags gui -ldflags "-H windowsgui" -o build\PlexiChat-GUI.exe .
    if errorlevel 1 (
        echo All builds failed!
        pause
        exit /b 1
    )
)

REM Build CLI version
echo Building CLI application...
go build -o build\PlexiChat-CLI.exe .

if errorlevel 1 (
    echo CLI build failed!
    pause
    exit /b 1
)

echo.
echo ========================================
echo Build Complete!
echo ========================================
echo GUI Application: build\PlexiChat-GUI.exe
echo CLI Application: build\PlexiChat-CLI.exe
echo.
echo To run GUI: .\build\PlexiChat-GUI.exe
echo To run CLI: .\build\PlexiChat-CLI.exe gui
echo.
pause
