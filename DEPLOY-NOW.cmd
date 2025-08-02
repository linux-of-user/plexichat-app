@echo off
echo ========================================
echo PlexiChat Desktop - DEPLOY NOW!
echo ========================================
echo.

echo 🚀 Ready to deploy PlexiChat Desktop to GitHub!
echo.

echo Step 1: Initialize Git repository...
git init
if errorlevel 1 (
    echo ❌ Git init failed - make sure Git is installed
    pause
    exit /b 1
)
echo ✅ Git repository initialized

echo.
echo Step 2: Add all files...
git add .
if errorlevel 1 (
    echo ❌ Git add failed
    pause
    exit /b 1
)
echo ✅ Files added to Git

echo.
echo Step 3: Create initial commit...
git commit -m "feat: PlexiChat Desktop v2.0.0-alpha - The Phoenix Release

🚀 Complete desktop application for PlexiChat
✨ Beautiful CLI with comprehensive features  
🎨 GUI support with Fyne (when CGO available)
🔐 Real API integration with authentication
💬 Chat, file sharing, and admin features
📱 Cross-platform compatibility
🛡️ Enterprise-grade error handling

Features:
- Complete CLI with all PlexiChat API features
- Native GUI support (requires CGO)
- Real-time chat and messaging
- File upload/download operations
- User authentication and management
- Admin operations and monitoring
- Cross-platform builds
- Professional documentation"

if errorlevel 1 (
    echo ❌ Git commit failed
    pause
    exit /b 1
)
echo ✅ Initial commit created

echo.
echo Step 4: Create GitHub repository...
gh repo create plexichat-desktop --public --description "🚀 Modern desktop application for PlexiChat with CLI and GUI support"
if errorlevel 1 (
    echo ❌ GitHub repo creation failed - make sure GitHub CLI is authenticated
    echo Run: gh auth login
    pause
    exit /b 1
)
echo ✅ GitHub repository created

echo.
echo Step 5: Set up remote and push...
git branch -M main
git remote add origin https://github.com/%USERNAME%/plexichat-desktop.git
git push -u origin main
if errorlevel 1 (
    echo ❌ Git push failed
    pause
    exit /b 1
)
echo ✅ Code pushed to GitHub

echo.
echo Step 6: Create GitHub release...
gh release create v2.0.0-alpha ^
  --title "🚀 PlexiChat Desktop v2.0.0-alpha - The Phoenix Release" ^
  --notes "## 🎉 The Phoenix Release

**PlexiChat Desktop rises from the ashes with a complete rewrite!**

### ✨ What's New
- 🖥️ **Modern CLI** with comprehensive PlexiChat API integration
- 🎨 **Native GUI** support (when CGO/C compiler available)  
- 🔐 **Complete authentication** with login, logout, and session management
- 💬 **Real-time chat** with message sending and receiving
- 📁 **File operations** with upload, download, and management
- 👥 **User management** with admin capabilities
- 🌐 **Web interface** fallback option
- 🛡️ **Enterprise-grade** error handling and validation

### 🚀 Quick Start
1. Download \`plexichat.exe\`
2. Run: \`plexichat.exe --help\`
3. Login: \`plexichat.exe auth login\`
4. Chat: \`plexichat.exe chat send --message \"Hello!\"\`

### 🎨 GUI Support
- Use: \`plexichat.exe gui\`
- Requires CGO and C compiler for building
- See SETUP-GUIDE.md for GUI setup instructions

### 📖 Documentation
- See README.md for detailed usage
- Check SETUP-GUIDE.md for GUI setup
- Review CONTRIBUTING.md for development

### 🔧 System Requirements
- Windows 10/11 (64-bit)
- For GUI: C compiler (GCC/MinGW/MSVC)
- For development: Go 1.21+

---
**This is a prerelease** - please report any issues!" ^
  --prerelease ^
  plexichat.exe

if errorlevel 1 (
    echo ❌ GitHub release creation failed
    pause
    exit /b 1
)
echo ✅ GitHub release created

echo.
echo Step 7: Add additional files to release...
if exist test-simple.exe (
    gh release upload v2.0.0-alpha test-simple.exe
    echo ✅ Test executable added to release
)

echo.
echo ========================================
echo 🎉 DEPLOYMENT SUCCESSFUL!
echo ========================================
echo.
echo Your PlexiChat Desktop is now live on GitHub!
echo.
echo 📍 Repository: https://github.com/%USERNAME%/plexichat-desktop
echo 📦 Release: https://github.com/%USERNAME%/plexichat-desktop/releases/tag/v2.0.0-alpha
echo.
echo 🚀 What's deployed:
echo   ✅ Complete source code
echo   ✅ Professional documentation
echo   ✅ Working executables
echo   ✅ GitHub release with binaries
echo   ✅ Ready for users and contributors
echo.
echo 🎯 Next steps:
echo   1. Share the repository with your team
echo   2. Test with real PlexiChat server
echo   3. Gather user feedback
echo   4. Plan next release features
echo.
echo ========================================
echo 🔥 THE PHOENIX HAS RISEN! 🔥
echo ========================================
pause
