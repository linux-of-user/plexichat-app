@echo off
setlocal enabledelayedexpansion

echo ========================================
echo PlexiChat Client - Windows Installer
echo ========================================
echo.

REM Check for admin privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Running with administrator privileges...
    set ADMIN_MODE=1
) else (
    echo Running without administrator privileges...
    set ADMIN_MODE=0
)

REM Set installation directories
if %ADMIN_MODE%==1 (
    set INSTALL_DIR=%ProgramFiles%\PlexiChat
    set CONFIG_DIR=%ALLUSERSPROFILE%\PlexiChat
) else (
    set INSTALL_DIR=%LOCALAPPDATA%\PlexiChat
    set CONFIG_DIR=%USERPROFILE%\.plexichat-app
)

echo Installation directory: %INSTALL_DIR%
echo Configuration directory: %CONFIG_DIR%
echo.

REM Create directories
echo [1/6] Creating directories...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%CONFIG_DIR%" mkdir "%CONFIG_DIR%"
echo ✅ Directories created

REM Download latest release if not provided
echo [2/6] Checking for binaries...
if not exist "plexichat-cli-windows-amd64.exe" (
    echo Downloading latest release...
    powershell -Command "& {
        $repo = 'linux-of-user/plexichat-app'
        $release = Invoke-RestMethod -Uri \"https://api.github.com/repos/$repo/releases/latest\"
        $cliAsset = $release.assets | Where-Object { $_.name -eq 'plexichat-cli-windows-amd64.exe' }
        $guiAsset = $release.assets | Where-Object { $_.name -eq 'plexichat-gui-windows-amd64.exe' }
        
        if ($cliAsset) {
            Write-Host 'Downloading CLI...'
            Invoke-WebRequest -Uri $cliAsset.browser_download_url -OutFile 'plexichat-cli-windows-amd64.exe'
        }
        
        if ($guiAsset) {
            Write-Host 'Downloading GUI...'
            Invoke-WebRequest -Uri $guiAsset.browser_download_url -OutFile 'plexichat-gui-windows-amd64.exe'
        }
    }"
    
    if %ERRORLEVEL% NEQ 0 (
        echo ❌ Failed to download binaries
        echo Please download them manually from:
        echo https://github.com/linux-of-user/plexichat-app/releases/latest
        pause
        exit /b 1
    )
)

REM Copy binaries
echo [3/6] Installing binaries...
if exist "plexichat-cli-windows-amd64.exe" (
    copy "plexichat-cli-windows-amd64.exe" "%INSTALL_DIR%\plexichat-cli.exe" >nul
    echo ✅ CLI installed
) else (
    echo ❌ CLI binary not found
)

if exist "plexichat-gui-windows-amd64.exe" (
    copy "plexichat-gui-windows-amd64.exe" "%INSTALL_DIR%\plexichat-gui.exe" >nul
    echo ✅ GUI installed
) else (
    echo ❌ GUI binary not found
)

REM Add to PATH
echo [4/6] Adding to PATH...
if %ADMIN_MODE%==1 (
    REM System-wide PATH
    for /f "tokens=2*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2^>nul') do set "SYSTEM_PATH=%%B"
    echo !SYSTEM_PATH! | findstr /C:"%INSTALL_DIR%" >nul
    if !ERRORLEVEL! NEQ 0 (
        reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH /t REG_EXPAND_SZ /d "!SYSTEM_PATH!;%INSTALL_DIR%" /f >nul
        echo ✅ Added to system PATH
    ) else (
        echo ✅ Already in system PATH
    )
) else (
    REM User PATH
    for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%B"
    if "!USER_PATH!"=="" set "USER_PATH=%PATH%"
    echo !USER_PATH! | findstr /C:"%INSTALL_DIR%" >nul
    if !ERRORLEVEL! NEQ 0 (
        reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!USER_PATH!;%INSTALL_DIR%" /f >nul
        echo ✅ Added to user PATH
    ) else (
        echo ✅ Already in user PATH
    )
)

REM Create desktop shortcuts
echo [5/6] Creating shortcuts...
powershell -Command "& {
    $WshShell = New-Object -comObject WScript.Shell
    
    # CLI shortcut
    $Shortcut = $WshShell.CreateShortcut('%USERPROFILE%\Desktop\PlexiChat CLI.lnk')
    $Shortcut.TargetPath = '%INSTALL_DIR%\plexichat-cli.exe'
    $Shortcut.WorkingDirectory = '%CONFIG_DIR%'
    $Shortcut.Description = 'PlexiChat Command Line Interface'
    $Shortcut.Save()
    
    # GUI shortcut
    $Shortcut = $WshShell.CreateShortcut('%USERPROFILE%\Desktop\PlexiChat.lnk')
    $Shortcut.TargetPath = '%INSTALL_DIR%\plexichat-gui.exe'
    $Shortcut.WorkingDirectory = '%CONFIG_DIR%'
    $Shortcut.Description = 'PlexiChat Desktop Application'
    $Shortcut.Save()
    
    # Start Menu shortcuts
    $StartMenuDir = '%APPDATA%\Microsoft\Windows\Start Menu\Programs\PlexiChat'
    if (!(Test-Path '$StartMenuDir')) { New-Item -ItemType Directory -Path '$StartMenuDir' -Force }
    
    $Shortcut = $WshShell.CreateShortcut('$StartMenuDir\PlexiChat CLI.lnk')
    $Shortcut.TargetPath = '%INSTALL_DIR%\plexichat-cli.exe'
    $Shortcut.WorkingDirectory = '%CONFIG_DIR%'
    $Shortcut.Save()
    
    $Shortcut = $WshShell.CreateShortcut('$StartMenuDir\PlexiChat.lnk')
    $Shortcut.TargetPath = '%INSTALL_DIR%\plexichat-gui.exe'
    $Shortcut.WorkingDirectory = '%CONFIG_DIR%'
    $Shortcut.Save()
}"
echo ✅ Shortcuts created

REM Initialize configuration
echo [6/6] Initializing configuration...
cd /d "%CONFIG_DIR%"
"%INSTALL_DIR%\plexichat-cli.exe" config init --force >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo ✅ Configuration initialized
) else (
    echo ⚠️  Configuration will be created on first run
)

echo.
echo ========================================
echo 🎉 Installation Complete!
echo ========================================
echo.
echo PlexiChat Client has been installed successfully!
echo.
echo Installation Details:
echo   📁 Install Directory: %INSTALL_DIR%
echo   ⚙️  Config Directory:  %CONFIG_DIR%
echo   🖥️  Desktop Shortcuts: Created
echo   📋 Start Menu:        Created
echo   🛤️  PATH:             Updated
echo.
echo Quick Start:
echo   1. Open Command Prompt or PowerShell
echo   2. Run: plexichat-cli config set url "http://your-server:8000"
echo   3. Run: plexichat-cli chat
echo   4. Or double-click "PlexiChat" on your desktop
echo.
echo 📚 Documentation: https://github.com/linux-of-user/plexichat-app
echo 🐛 Issues: https://github.com/linux-of-user/plexichat-app/issues
echo.
echo 🚀 Ready to use PlexiChat!
echo.
pause
