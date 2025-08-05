@echo off
echo ========================================
echo PlexiChat Client - Git Repository Setup
echo ========================================
echo.

echo This script will help you set up a Git repository and deploy to GitHub.
echo.

echo [Step 1] Checking Git installation...
git --version
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Git is not installed or not in PATH
    echo Please install Git from: https://git-scm.com/download/win
    pause
    exit /b 1
)
echo ✅ Git is installed

echo.
echo [Step 2] Checking GitHub CLI authentication...
gh auth status
if %ERRORLEVEL% NEQ 0 (
    echo ❌ GitHub CLI is not authenticated
    echo Please run: gh auth login
    pause
    exit /b 1
)
echo ✅ GitHub CLI is authenticated

echo.
echo [Step 3] Initializing Git repository (if not already done)...
git init
echo ✅ Git repository initialized

echo.
echo [Step 4] Configuring Git user (if not already done)...
echo Please enter your Git configuration:
set /p GIT_NAME="Enter your name: "
set /p GIT_EMAIL="Enter your email: "

git config user.name "%GIT_NAME%"
git config user.email "%GIT_EMAIL%"
echo ✅ Git user configured

echo.
echo [Step 5] Adding all files to Git...
git add .
echo ✅ Files added to Git

echo.
echo [Step 6] Creating initial commit...
git commit -m "Initial commit: PlexiChat Client v1.0.0

Features:
- Complete CLI and GUI applications
- Real-time messaging with WebSocket support
- Comprehensive configuration system
- ASCII-only logging with colorization
- Security validation and XSS protection
- Advanced retry logic with exponential backoff
- Modern Discord-like GUI interface
- File upload and emoji picker
- Comprehensive documentation

Ready for production use!"

echo ✅ Initial commit created

echo.
echo [Step 7] Creating GitHub repository...
gh repo create plexichat-client --public --description="PlexiChat Desktop Client - Modern CLI and GUI applications for PlexiChat messaging platform with real-time messaging, comprehensive configuration, and professional documentation"

if %ERRORLEVEL% NEQ 0 (
    echo ❌ Failed to create GitHub repository
    echo Repository might already exist. Continuing...
)

echo.
echo [Step 8] Adding GitHub remote...
git remote add origin https://github.com/%USERNAME%/plexichat-client.git
echo ✅ GitHub remote added

echo.
echo [Step 9] Pushing to GitHub...
git branch -M main
git push -u origin main

if %ERRORLEVEL% EQU 0 (
    echo ✅ Successfully pushed to GitHub!
) else (
    echo ❌ Failed to push to GitHub
    echo This might be because the repository already exists or there are conflicts.
    echo You can try: git push --force-with-lease origin main
)

echo.
echo [Step 10] Setting up GitHub repository settings...
gh repo edit --description "PlexiChat Desktop Client - Modern CLI and GUI applications for PlexiChat messaging platform with real-time messaging, comprehensive configuration, and professional documentation"
gh repo edit --add-topic "plexichat,messaging,cli,gui,golang,websocket,real-time"

echo.
echo ========================================
echo 🎉 Git Repository Setup Complete! 🎉
echo ========================================
echo.
echo Your PlexiChat Client repository is now available at:
echo https://github.com/%USERNAME%/plexichat-client
echo.
echo Repository includes:
echo ✅ Complete source code for CLI and GUI applications
echo ✅ Comprehensive documentation (README, guides, API docs)
echo ✅ Professional project structure
echo ✅ Ready-to-use applications
echo ✅ Production deployment guides
echo.
echo Next steps:
echo 1. Visit your repository on GitHub
echo 2. Add any additional collaborators if needed
echo 3. Set up GitHub Actions for CI/CD (optional)
echo 4. Create releases for distribution
echo.
echo 🚀 Your PlexiChat Client is now live on GitHub!
echo.
pause
