# 🚀 PlexiChat Desktop Client

A **professional, feature-rich desktop client** for the PlexiChat messaging platform with modern CLI and GUI interfaces, real-time messaging, and comprehensive configuration management.

![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)
![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen)

## 📱 Applications

- **`plexichat-cli.exe`** - Full-featured command-line interface
- **`plexichat-gui.exe`** - Modern graphical messaging interface

## ✨ Key Features

### 🖥️ **Dual Interface**
- **CLI Application**: Interactive commands, configuration management, real-time chat
- **GUI Application**: Modern messaging interface with professional design

### 💬 **Real-time Communication**
- **WebSocket messaging** with automatic reconnection
- **Live message delivery** and typing indicators
- **Channel management** and user presence tracking

### 🔧 **Advanced Configuration**
- **YAML/JSON configuration** files with validation
- **Environment variable** overrides (`PLEXICHAT_*`)
- **Command-line flags** for all settings
- **Configuration management** commands (init, show, set, validate)

### 🔒 **Security & Validation**
- **Input validation** with XSS protection
- **Password strength** validation with common password detection
- **File upload security** with type and size validation
- **Secure authentication** with JWT token support

### 📝 **Professional Logging**
- **ASCII-only output** with configurable colorization
- **Multiple log levels** (DEBUG, INFO, WARN, ERROR, FATAL)
- **Custom logger instances** with prefixes and formatting
- **Error handling** and recovery mechanisms

### GUI Features
- **Modern Fyne-based interface** with professional design
- **Dark/Light theme support** with user preferences
- **User avatars** with automatic generation and color coding
- **Emoji picker** with 100+ emojis in organized categories
- **File drag & drop** support with upload confirmation
- **Real-time notifications** with desktop integration

### CLI Features
- **Interactive chat mode** with real-time messaging
- **Channel management** (list, join, leave, create)
- **User management** and authentication
- **Configuration management** with YAML support
- **Comprehensive help system** and command completion

### Security Features
- **2FA/MFA support** (TOTP, SMS, Email, Hardware keys)
- **Input validation** with XSS protection
- **Secure authentication** with token management
- **Password strength validation**
- **File upload security** with type and size validation

## 🚀 Quick Start

### 📦 Download & Run
1. **Download** the latest release from [Releases](https://github.com/linux-of-user/plexichat-app/releases)
2. **Extract** the archive
3. **Run** the application:
   ```bash
   # CLI Application
   ./plexichat-cli.exe config init
   ./plexichat-cli.exe config set url "http://localhost:8000"
   ./plexichat-cli.exe chat

   # GUI Application
   ./plexichat-gui.exe
   ```

### 🛠️ Build from Source
**Prerequisites:** Go 1.19+, CGO enabled (for GUI)

```bash
# Clone repository
git clone https://github.com/linux-of-user/plexichat-app.git
cd plexichat-app

# Build CLI
go build -o plexichat-cli.exe plexichat-cli.go

# Build GUI (requires CGO)
set CGO_ENABLED=1
go build -o plexichat-gui.exe plexichat-gui.go
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

- 📖 [Documentation](https://github.com/linux-of-user/plexichat-app/wiki)
- 🐛 [Report Issues](https://github.com/linux-of-user/plexichat-app/issues)
- 💬 [Discussions](https://github.com/linux-of-user/plexichat-app/discussions)

---

**Made with ❤️ for the PlexiChat community**
