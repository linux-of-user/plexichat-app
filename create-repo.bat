@echo off
echo Creating PlexiChat Desktop GitHub Repository...

REM Initialize git repository
git init
git add .
git commit -m "Initial commit: PlexiChat Desktop v2.0.0-alpha"

REM Create GitHub repository
gh repo create plexichat-desktop --public --description "Beautiful, modern desktop application for PlexiChat with native GUI and real-time messaging"

REM Push to GitHub
git branch -M main
git remote add origin https://github.com/dboyn/plexichat-desktop.git
git push -u origin main

echo Repository created successfully!
echo Creating release...

REM Create release with builds
gh release create v2.0.0-alpha ^
  --title "PlexiChat Desktop v2.0.0-alpha - The Phoenix Release" ^
  --notes "🚀 **The Phoenix Release** - PlexiChat Desktop rises from the ashes with a completely rewritten, beautiful native GUI!

## ✨ What's New

### 🎨 **Beautiful Native Interface**
- Modern Fyne-based GUI with professional design
- Dark/Light theme support with user preferences  
- User avatars with automatic generation and color coding
- Emoji picker with 100+ emojis in organized categories
- File drag & drop support with upload confirmation

### 🔐 **Complete Authentication System**
- Real API integration with PlexiChat server
- 2FA/MFA support (TOTP, SMS, Email, Hardware keys)
- Session management with auto-login and token persistence
- Secure logout with complete session cleanup
- User registration with validation and confirmation

### 💬 **Real-Time Communication**
- Live message sending via PlexiChat API
- Message history with timestamps and user avatars
- Group management with creation and member handling
- File sharing with drag & drop support
- Smart error handling with retry logic

### ⚙️ **Advanced Features**
- Comprehensive settings panel with preferences
- Keyboard shortcuts for power users (F1 help, F11 fullscreen)
- Status monitoring with connection indicators
- Cross-platform builds for Windows, macOS, and Linux

## 🚀 Quick Start

1. Download `PlexiChat-GUI-windows-amd64.exe` for the GUI application
2. Or download `PlexiChat-CLI-windows-amd64.exe` and run `./PlexiChat-CLI.exe gui`
3. Enter your PlexiChat server details and login!

## ⌨️ Keyboard Shortcuts

- **Enter** - Send message
- **F1** - Show help and shortcuts  
- **F11** - Toggle fullscreen
- **Escape** - Close dialogs

---

**This is a prerelease** - Please report any issues you encounter!" ^
  --prerelease ^
  build/PlexiChat-GUI-windows-amd64.exe ^
  build/PlexiChat-CLI-windows-amd64.exe

echo Release created successfully!
echo.
echo Repository: https://github.com/dboyn/plexichat-desktop
echo Release: https://github.com/dboyn/plexichat-desktop/releases/tag/v2.0.0-alpha
echo.
pause
