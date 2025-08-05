@echo off
echo Building PlexiChat CLI...

if not exist build mkdir build

echo Building CLI client...
go build -o build\plexichat.exe .\cmd\plexichat

if %ERRORLEVEL% EQU 0 (
    echo ✅ CLI build successful
    echo Testing CLI...
    build\plexichat.exe version
    echo.
    echo CLI is ready! Try:
    echo   build\plexichat.exe help
    echo   build\plexichat.exe health
    echo   build\plexichat.exe test
) else (
    echo ❌ CLI build failed
    exit /b 1
)
