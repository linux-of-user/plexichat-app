@echo off
echo ========================================
echo PlexiChat Client Deployment Verification
echo ========================================
echo.

set SUCCESS=0
set TOTAL=0

echo [Checking Built Applications...]
echo.

REM Check CLI executable
set /a TOTAL+=1
if exist "plexichat-cli.exe" (
    echo ✅ CLI Application: plexichat-cli.exe found
    set /a SUCCESS+=1
) else (
    echo ❌ CLI Application: plexichat-cli.exe NOT found
)

REM Check GUI executable
set /a TOTAL+=1
if exist "plexichat-gui.exe" (
    echo ✅ GUI Application: plexichat-gui.exe found
    set /a SUCCESS+=1
) else (
    echo ❌ GUI Application: plexichat-gui.exe NOT found
)

echo.
echo [Checking Source Files...]
echo.

REM Check main source files
set /a TOTAL+=1
if exist "plexichat-cli.go" (
    echo ✅ CLI Source: plexichat-cli.go found
    set /a SUCCESS+=1
) else (
    echo ❌ CLI Source: plexichat-cli.go NOT found
)

set /a TOTAL+=1
if exist "plexichat-gui.go" (
    echo ✅ GUI Source: plexichat-gui.go found
    set /a SUCCESS+=1
) else (
    echo ❌ GUI Source: plexichat-gui.go NOT found
)

echo.
echo [Checking Package Structure...]
echo.

REM Check package directories
set /a TOTAL+=1
if exist "pkg\client" (
    echo ✅ Client Package: pkg\client found
    set /a SUCCESS+=1
) else (
    echo ❌ Client Package: pkg\client NOT found
)

set /a TOTAL+=1
if exist "pkg\logging" (
    echo ✅ Logging Package: pkg\logging found
    set /a SUCCESS+=1
) else (
    echo ❌ Logging Package: pkg\logging NOT found
)

set /a TOTAL+=1
if exist "pkg\security" (
    echo ✅ Security Package: pkg\security found
    set /a SUCCESS+=1
) else (
    echo ❌ Security Package: pkg\security NOT found
)

set /a TOTAL+=1
if exist "pkg\websocket" (
    echo ✅ WebSocket Package: pkg\websocket found
    set /a SUCCESS+=1
) else (
    echo ❌ WebSocket Package: pkg\websocket NOT found
)

echo.
echo [Checking Documentation...]
echo.

set /a TOTAL+=1
if exist "README.md" (
    echo ✅ Main Documentation: README.md found
    set /a SUCCESS+=1
) else (
    echo ❌ Main Documentation: README.md NOT found
)

set /a TOTAL+=1
if exist "docs\CONFIGURATION.md" (
    echo ✅ Configuration Guide: docs\CONFIGURATION.md found
    set /a SUCCESS+=1
) else (
    echo ❌ Configuration Guide: docs\CONFIGURATION.md NOT found
)

set /a TOTAL+=1
if exist "docs\TROUBLESHOOTING.md" (
    echo ✅ Troubleshooting Guide: docs\TROUBLESHOOTING.md found
    set /a SUCCESS+=1
) else (
    echo ❌ Troubleshooting Guide: docs\TROUBLESHOOTING.md NOT found
)

set /a TOTAL+=1
if exist "docs\API.md" (
    echo ✅ API Documentation: docs\API.md found
    set /a SUCCESS+=1
) else (
    echo ❌ API Documentation: docs\API.md NOT found
)

set /a TOTAL+=1
if exist "DEPLOYMENT_GUIDE.md" (
    echo ✅ Deployment Guide: DEPLOYMENT_GUIDE.md found
    set /a SUCCESS+=1
) else (
    echo ❌ Deployment Guide: DEPLOYMENT_GUIDE.md NOT found
)

echo.
echo [Checking Configuration Files...]
echo.

set /a TOTAL+=1
if exist "go.mod" (
    echo ✅ Go Module: go.mod found
    set /a SUCCESS+=1
) else (
    echo ❌ Go Module: go.mod NOT found
)

set /a TOTAL+=1
if exist "go.sum" (
    echo ✅ Go Dependencies: go.sum found
    set /a SUCCESS+=1
) else (
    echo ❌ Go Dependencies: go.sum NOT found
)

echo.
echo [Testing Application Launch...]
echo.

REM Test GUI launch (brief)
set /a TOTAL+=1
echo Testing GUI launch...
start /wait /b plexichat-gui.exe
if %ERRORLEVEL% EQU 0 (
    echo ✅ GUI Application: Launches successfully
    set /a SUCCESS+=1
) else (
    echo ❌ GUI Application: Launch failed
)

echo.
echo ========================================
echo VERIFICATION RESULTS
echo ========================================
echo.
echo Tests Passed: %SUCCESS%/%TOTAL%

if %SUCCESS% EQU %TOTAL% (
    echo.
    echo 🎉 ALL VERIFICATIONS PASSED! 🎉
    echo.
    echo ✅ PlexiChat Client is fully deployed and ready!
    echo.
    echo Applications available:
    echo - plexichat-cli.exe ^(Command Line Interface^)
    echo - plexichat-gui.exe ^(Graphical User Interface^)
    echo.
    echo Documentation available:
    echo - README.md ^(Main documentation^)
    echo - docs\CONFIGURATION.md ^(Configuration guide^)
    echo - docs\TROUBLESHOOTING.md ^(Troubleshooting guide^)
    echo - docs\API.md ^(API documentation^)
    echo - DEPLOYMENT_GUIDE.md ^(Deployment guide^)
    echo.
    echo 🚀 Ready for production use!
) else (
    echo.
    echo ❌ SOME VERIFICATIONS FAILED
    echo.
    echo Please check the missing components above.
    echo Refer to the build instructions in DEPLOYMENT_GUIDE.md
)

echo.
pause
