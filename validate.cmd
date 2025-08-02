@echo off
echo ========================================
echo PlexiChat Desktop Validation Suite
echo ========================================
echo.

set TESTS_PASSED=0
set TESTS_FAILED=0

echo 🔍 Validating project structure...

REM Check essential files
if exist main.go (
    echo ✅ main.go exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ main.go missing
    set /a TESTS_FAILED+=1
)

if exist go.mod (
    echo ✅ go.mod exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ go.mod missing
    set /a TESTS_FAILED+=1
)

if exist README.md (
    echo ✅ README.md exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ README.md missing
    set /a TESTS_FAILED+=1
)

if exist cmd\root.go (
    echo ✅ cmd\root.go exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ cmd\root.go missing
    set /a TESTS_FAILED+=1
)

if exist cmd\gui_launcher.go (
    echo ✅ cmd\gui_launcher.go exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ cmd\gui_launcher.go missing
    set /a TESTS_FAILED+=1
)

if exist pkg\client (
    echo ✅ pkg\client directory exists
    set /a TESTS_PASSED+=1
) else (
    echo ❌ pkg\client directory missing
    set /a TESTS_FAILED+=1
)

echo.
echo 🧪 Testing Go environment...

go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Go not installed or not in PATH
    set /a TESTS_FAILED+=1
) else (
    echo ✅ Go is available
    set /a TESTS_PASSED+=1
)

echo.
echo 🏗️ Testing build...

go build -o test-build.exe . >nul 2>&1
if errorlevel 1 (
    echo ❌ Build failed
    set /a TESTS_FAILED+=1
) else (
    echo ✅ Build successful
    set /a TESTS_PASSED+=1
    
    REM Test execution
    test-build.exe --version >nul 2>&1
    if errorlevel 1 (
        echo ❌ Execution failed
        set /a TESTS_FAILED+=1
    ) else (
        echo ✅ Execution successful
        set /a TESTS_PASSED+=1
    )
    
    REM Clean up
    del test-build.exe
)

echo.
echo 📋 Testing commands...

go run . --help >nul 2>&1
if errorlevel 1 (
    echo ❌ Help command failed
    set /a TESTS_FAILED+=1
) else (
    echo ✅ Help command works
    set /a TESTS_PASSED+=1
)

echo.
echo ========================================
echo Validation Results
echo ========================================
echo.
echo Tests Passed: %TESTS_PASSED%
echo Tests Failed: %TESTS_FAILED%

if %TESTS_FAILED% EQU 0 (
    echo.
    echo 🎉 ALL TESTS PASSED!
    echo ✅ PlexiChat Desktop is ready for deployment
    echo.
    echo Next steps:
    echo 1. Run: build-simple.cmd
    echo 2. Follow: SETUP-AND-DEPLOY.md
    echo 3. Deploy to GitHub
    echo.
) else (
    echo.
    echo ❌ %TESTS_FAILED% tests failed
    echo Please fix issues before deployment
    echo.
)

echo ========================================
pause
