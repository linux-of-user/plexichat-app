# 🚀 PlexiChat Desktop Client

A **beautiful, modern desktop application** for PlexiChat with native GUI, real-time messaging, and comprehensive API integration.

![PlexiChat Desktop](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Version](https://img.shields.io/badge/Version-2.0.0--alpha-orange)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ Features

### 🎨 **Beautiful Native GUI**
- **Modern Fyne-based interface** with professional design
- **Dark/Light theme support** with user preferences
- **User avatars** with automatic generation and color coding
- **Emoji picker** with 100+ emojis in organized categories
- **File drag & drop** support with upload confirmation
- **Real-time notifications** with desktop integration

### 🔐 **Complete Authentication**
- **Real API integration** with PlexiChat server
- **2FA/MFA support** (TOTP, SMS, Email, Hardware keys)
- **Session management** with auto-login and token persistence
- **Secure logout** with complete session cleanup
- **User registration** with validation and confirmation

### 💬 **Real-Time Communication**
- **Live message sending** via PlexiChat API
- **Message history** with timestamps and user avatars
- **Group management** with creation and member handling
- **Typing indicators** and message status
- **File sharing** with drag & drop support

### ⚙️ **Advanced Features**
- **Comprehensive settings** panel with preferences
- **Keyboard shortcuts** for power users (F1 help, F11 fullscreen)
- **Error handling** with smart retry logic and user-friendly messages
- **Status monitoring** with connection indicators
- **Cross-platform** builds for Windows, macOS, and Linux

## 🚀 Quick Start

### Download & Run
1. Download the latest release from [Releases](https://github.com/yourusername/plexichat-desktop/releases)
2. Extract the archive
3. Run `PlexiChat-GUI.exe` (Windows) or equivalent for your platform
4. Enter your PlexiChat server details and login!

### From Source
```bash
git clone https://github.com/yourusername/plexichat-desktop
cd plexichat-desktop
go build -o PlexiChat-CLI.exe .
./PlexiChat-CLI.exe gui
```

## 🎯 Usage

### GUI Application
- **Windows**: `PlexiChat-GUI.exe`
- **macOS**: `PlexiChat-GUI.app`
- **Linux**: `PlexiChat-GUI`

### CLI Commands
```bash
# Launch GUI
./PlexiChat-CLI gui

# Login via CLI
./PlexiChat-CLI auth login --username your-username

# Send message via CLI
./PlexiChat-CLI chat send --message "Hello!" --room general

# Show help
./PlexiChat-CLI --help
```

## ⌨️ Keyboard Shortcuts

- **Enter** - Send message
- **F1** - Show help and shortcuts
- **F11** - Toggle fullscreen
- **Escape** - Close dialogs
- **Ctrl+Enter** - New line in message

## 🛠️ Development

### Prerequisites
- Go 1.21+
- CGO enabled (for GUI builds)
- C compiler (GCC/MinGW on Windows)

### Building
```bash
# GUI Application
fyne package -name PlexiChat -icon icon.png

# CLI Application  
go build -o PlexiChat-CLI.exe .

# Cross-platform builds
./build-release.bat  # Windows
./build-release.sh   # Linux/macOS
```

## 📦 Release Builds

This project uses semantic versioning with a clever twist:
- **Major.Minor.Patch-Stage** (e.g., `2.0.0-alpha`)
- **Stages**: `alpha` → `beta` → `rc` → `stable`
- **Special builds**: `nightly`, `experimental`, `hotfix`

## 🎨 Screenshots

*Coming soon - beautiful screenshots of the modern interface!*

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 [Documentation](https://github.com/yourusername/plexichat-desktop/wiki)
- 🐛 [Report Issues](https://github.com/yourusername/plexichat-desktop/issues)
- 💬 [Discussions](https://github.com/yourusername/plexichat-desktop/discussions)

---

**Made with ❤️ for the PlexiChat community**
