@echo off
echo ========================================
echo PlexiChat Desktop Builder
echo ========================================
echo.

REM Clean up
if exist build rmdir /s /q build
mkdir build

echo Step 1: Testing Go environment...
go version
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    pause
    exit /b 1
)
echo ✓ Go is available

echo.
echo Step 2: Testing basic build...
go build -o build\plexichat-cli.exe .
if errorlevel 1 (
    echo ERROR: Basic build failed
    pause
    exit /b 1
)
echo ✓ Basic CLI build successful

echo.
echo Step 3: Testing CLI functionality...
build\plexichat-cli.exe --version
if errorlevel 1 (
    echo ERROR: CLI execution failed
    pause
    exit /b 1
)
echo ✓ CLI execution successful

echo.
echo Step 4: Testing dependencies...
go mod download
if errorlevel 1 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)
echo ✓ Dependencies downloaded

echo.
echo Step 5: Checking CGO status...
for /f "tokens=*" %%i in ('go env CGO_ENABLED') do set CGO_STATUS=%%i
echo CGO_ENABLED: %CGO_STATUS%

if "%CGO_STATUS%"=="0" (
    echo WARNING: CGO is disabled - GUI build will fail
    echo Setting CGO_ENABLED=1...
    set CGO_ENABLED=1
)

echo.
echo Step 6: Testing C compiler...
gcc --version >nul 2>&1
if errorlevel 1 (
    echo WARNING: GCC not found - trying other compilers...
    cl >nul 2>&1
    if errorlevel 1 (
        echo WARNING: No C compiler found - GUI build will fail
        echo You can still use the CLI version
        goto :skip_gui
    ) else (
        echo ✓ MSVC compiler found
    )
) else (
    echo ✓ GCC compiler found
)

echo.
echo Step 7: Attempting GUI build...
set CGO_ENABLED=1
go build -tags gui -o build\plexichat-gui.exe .
if errorlevel 1 (
    echo WARNING: GUI build failed - this is expected without proper CGO setup
    echo See SETUP-GUIDE.md for CGO setup instructions
    goto :skip_gui
)
echo ✓ GUI build successful

:skip_gui
echo.
echo ========================================
echo Build Summary
echo ========================================
echo.
echo Built files:
if exist build\*.exe dir build\*.exe

echo.
echo Usage:
echo   CLI: .\build\plexichat-cli.exe --help
echo   GUI: .\build\plexichat-gui.exe (if available)
echo   GUI via CLI: .\build\plexichat-cli.exe gui
echo.

echo Testing CLI help...
build\plexichat-cli.exe --help
echo.

echo ========================================
echo Build Complete!
echo ========================================
pause
