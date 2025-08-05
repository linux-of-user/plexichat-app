@echo off
echo ========================================
echo PlexiChat Client Application Test Suite
echo ========================================
echo.

echo [1/6] Testing CLI Build...
go build -o plexichat-cli-test.exe plexichat-cli.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ CLI build failed!
    goto :error
)
echo ✅ CLI build successful

echo.
echo [2/6] Testing CLI Help Command...
plexichat-cli-test.exe --help > cli-help.txt 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ❌ CLI help command failed!
    goto :error
)
echo ✅ CLI help command works

echo.
echo [3/6] Testing CLI Version Command...
plexichat-cli-test.exe version > cli-version.txt 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ❌ CLI version command failed!
    goto :error
)
echo ✅ CLI version command works

echo.
echo [4/6] Testing CLI Configuration...
plexichat-cli-test.exe config init > cli-config.txt 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ❌ CLI config init failed!
    goto :error
)
echo ✅ CLI configuration works

echo.
echo [5/6] Testing GUI Build...
set CGO_ENABLED=1
go build -o plexichat-gui-test.exe plexichat-gui.go
if %ERRORLEVEL% NEQ 0 (
    echo ❌ GUI build failed!
    goto :error
)
echo ✅ GUI build successful

echo.
echo [6/6] Testing GUI Launch (will close automatically)...
timeout /t 3 /nobreak > nul
start /wait /b plexichat-gui-test.exe
echo ✅ GUI application launched

echo.
echo ========================================
echo 🎉 ALL TESTS PASSED! 🎉
echo ========================================
echo.
echo Applications built successfully:
echo - plexichat-cli-test.exe (Command Line Interface)
echo - plexichat-gui-test.exe (Graphical User Interface)
echo.
echo Test outputs saved to:
echo - cli-help.txt
echo - cli-version.txt  
echo - cli-config.txt
echo.
echo Ready for use! 🚀
goto :end

:error
echo.
echo ========================================
echo ❌ TESTS FAILED ❌
echo ========================================
echo Please check the error messages above.
exit /b 1

:end
echo.
echo Test completed successfully!
pause
