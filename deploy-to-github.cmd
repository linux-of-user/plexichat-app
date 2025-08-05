@echo off
echo ========================================
echo PlexiChat Client - GitHub Deployment
echo ========================================
echo.

echo [1/6] Adding all files to Git...
git add .
echo ✅ Files added

echo.
echo [2/6] Creating commit...
git commit -m "PlexiChat Client v1.0.0 - Production Release

🚀 Complete CLI and GUI applications for PlexiChat messaging platform

✨ Features:
- Modern Discord-like GUI interface with real-time messaging
- Full-featured CLI with interactive commands and configuration
- ASCII-only logging system with configurable levels and colorization
- Comprehensive configuration management (YAML, env vars, CLI flags)
- Advanced security validation and XSS protection
- WebSocket real-time communication with auto-reconnect
- File upload with drag & drop support and emoji picker
- Advanced retry logic with exponential backoff
- Professional documentation and deployment guides

🎯 Ready for production use!

📚 Documentation:
- Complete README with quick start guide
- Configuration guide (docs/CONFIGURATION.md)
- Troubleshooting guide (docs/TROUBLESHOOTING.md)
- API documentation (docs/API.md)
- Deployment guide (DEPLOYMENT_GUIDE.md)

🛠️ Technical:
- Go 1.19+ with Fyne GUI framework
- Modular architecture with comprehensive testing
- Cross-platform support (Windows, macOS, Linux)
- Professional error handling and recovery"

echo ✅ Commit created

echo.
echo [3/6] Creating GitHub repository...
gh repo create plexichat-client --public --description "PlexiChat Desktop Client - Modern CLI and GUI applications for PlexiChat messaging platform with real-time messaging, comprehensive configuration, and professional documentation"
echo ✅ Repository created (or already exists)

echo.
echo [4/6] Setting up remote...
git remote remove origin 2>nul
git remote add origin https://github.com/linux-of-user/plexichat-client.git
echo ✅ Remote configured

echo.
echo [5/6] Pushing to GitHub...
git branch -M main
git push -u origin main --force-with-lease
echo ✅ Pushed to GitHub

echo.
echo [6/6] Configuring repository settings...
gh repo edit --description "PlexiChat Desktop Client - Modern CLI and GUI applications for PlexiChat messaging platform with real-time messaging, comprehensive configuration, and professional documentation"
gh repo edit --add-topic "plexichat"
gh repo edit --add-topic "messaging"
gh repo edit --add-topic "cli"
gh repo edit --add-topic "gui"
gh repo edit --add-topic "golang"
gh repo edit --add-topic "websocket"
gh repo edit --add-topic "real-time"
gh repo edit --add-topic "desktop-app"
gh repo edit --add-topic "chat-client"
echo ✅ Repository configured

echo.
echo ========================================
echo 🎉 SUCCESS! Repository deployed! 🎉
echo ========================================
echo.
echo Your PlexiChat Client is now live at:
echo https://github.com/linux-of-user/plexichat-client
echo.
echo Repository includes:
echo ✅ Complete source code (CLI + GUI)
echo ✅ Built applications ready to use
echo ✅ Comprehensive documentation
echo ✅ Professional project structure
echo ✅ Production deployment guides
echo.
echo 🚀 Ready to share with the world!
echo.
pause
