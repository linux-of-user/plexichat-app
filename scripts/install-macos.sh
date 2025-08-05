#!/bin/bash

echo "========================================"
echo "PlexiChat Client - macOS Installer"
echo "========================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    echo -e "${YELLOW}Running as root - installing system-wide${NC}"
    INSTALL_DIR="/usr/local/bin"
    CONFIG_DIR="/etc/plexichat"
    SYSTEM_INSTALL=true
else
    echo -e "${BLUE}Running as user - installing to user directory${NC}"
    INSTALL_DIR="$HOME/.local/bin"
    CONFIG_DIR="$HOME/.plexichat-app"
    SYSTEM_INSTALL=false
fi

echo "Installation directory: $INSTALL_DIR"
echo "Configuration directory: $CONFIG_DIR"
echo

# Create directories
echo "[1/6] Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
echo -e "${GREEN}✅ Directories created${NC}"

# Download latest release if not provided
echo "[2/6] Checking for binaries..."
if [[ ! -f "plexichat-cli-macos-amd64" ]] || [[ ! -f "plexichat-gui-macos-amd64" ]]; then
    echo "Downloading latest release..."
    
    # Get latest release info
    REPO="linux-of-user/plexichat-app"
    API_URL="https://api.github.com/repos/$REPO/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        RELEASE_INFO=$(curl -s "$API_URL")
    else
        echo -e "${RED}❌ curl not found. Please install curl.${NC}"
        exit 1
    fi
    
    # Extract download URLs
    CLI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*plexichat-cli-macos-amd64[^"]*"' | cut -d'"' -f4)
    GUI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*plexichat-gui-macos-amd64[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$CLI_URL" ]]; then
        echo "Downloading CLI..."
        curl -L -o "plexichat-cli-macos-amd64" "$CLI_URL"
    fi
    
    if [[ -n "$GUI_URL" ]]; then
        echo "Downloading GUI..."
        curl -L -o "plexichat-gui-macos-amd64" "$GUI_URL"
    fi
    
    if [[ ! -f "plexichat-cli-macos-amd64" ]] && [[ ! -f "plexichat-gui-macos-amd64" ]]; then
        echo -e "${RED}❌ Failed to download binaries${NC}"
        echo "Please download them manually from:"
        echo "https://github.com/linux-of-user/plexichat-app/releases/latest"
        exit 1
    fi
fi

# Install binaries
echo "[3/6] Installing binaries..."
if [[ -f "plexichat-cli-macos-amd64" ]]; then
    cp "plexichat-cli-macos-amd64" "$INSTALL_DIR/plexichat-cli"
    chmod +x "$INSTALL_DIR/plexichat-cli"
    echo -e "${GREEN}✅ CLI installed${NC}"
else
    echo -e "${RED}❌ CLI binary not found${NC}"
fi

if [[ -f "plexichat-gui-macos-amd64" ]]; then
    cp "plexichat-gui-macos-amd64" "$INSTALL_DIR/plexichat-gui"
    chmod +x "$INSTALL_DIR/plexichat-gui"
    echo -e "${GREEN}✅ GUI installed${NC}"
else
    echo -e "${RED}❌ GUI binary not found${NC}"
fi

# Add to PATH
echo "[4/6] Adding to PATH..."
if [[ "$SYSTEM_INSTALL" == true ]]; then
    # System-wide installation - /usr/local/bin should already be in PATH
    echo -e "${GREEN}✅ System installation - already in PATH${NC}"
else
    # User installation - add to user's PATH
    SHELL_RC=""
    if [[ -n "$BASH_VERSION" ]]; then
        SHELL_RC="$HOME/.bash_profile"
        [[ ! -f "$SHELL_RC" ]] && SHELL_RC="$HOME/.bashrc"
    elif [[ -n "$ZSH_VERSION" ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ -f "$HOME/.profile" ]]; then
        SHELL_RC="$HOME/.profile"
    fi
    
    if [[ -n "$SHELL_RC" ]]; then
        if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
            echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
            echo -e "${GREEN}✅ Added to PATH in $SHELL_RC${NC}"
            echo -e "${YELLOW}⚠️  Please restart your terminal or run: source $SHELL_RC${NC}"
        else
            echo -e "${GREEN}✅ Already in PATH${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Could not detect shell. Please add $INSTALL_DIR to your PATH manually${NC}"
    fi
fi

# Create macOS app bundles
echo "[5/6] Creating application bundles..."
APPS_DIR="$HOME/Applications"
mkdir -p "$APPS_DIR"

# Create PlexiChat.app bundle
APP_BUNDLE="$APPS_DIR/PlexiChat.app"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

# Create Info.plist
cat > "$APP_BUNDLE/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>PlexiChat</string>
    <key>CFBundleIdentifier</key>
    <string>com.plexichat.client</string>
    <key>CFBundleName</key>
    <string>PlexiChat</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>PLXI</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.14</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.social-networking</string>
</dict>
</plist>
EOF

# Create launcher script
cat > "$APP_BUNDLE/Contents/MacOS/PlexiChat" << EOF
#!/bin/bash
cd "$CONFIG_DIR"
exec "$INSTALL_DIR/plexichat-gui"
EOF
chmod +x "$APP_BUNDLE/Contents/MacOS/PlexiChat"

# Create CLI Terminal app
CLI_APP_BUNDLE="$APPS_DIR/PlexiChat CLI.app"
mkdir -p "$CLI_APP_BUNDLE/Contents/MacOS"

cat > "$CLI_APP_BUNDLE/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>PlexiChat CLI</string>
    <key>CFBundleIdentifier</key>
    <string>com.plexichat.cli</string>
    <key>CFBundleName</key>
    <string>PlexiChat CLI</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.developer-tools</string>
</dict>
</plist>
EOF

cat > "$CLI_APP_BUNDLE/Contents/MacOS/PlexiChat CLI" << EOF
#!/bin/bash
osascript -e "tell application \"Terminal\" to do script \"cd '$CONFIG_DIR' && '$INSTALL_DIR/plexichat-cli' chat\""
EOF
chmod +x "$CLI_APP_BUNDLE/Contents/MacOS/PlexiChat CLI"

echo -e "${GREEN}✅ Application bundles created${NC}"

# Initialize configuration
echo "[6/6] Initializing configuration..."
cd "$CONFIG_DIR"
if "$INSTALL_DIR/plexichat-cli" config init --force >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Configuration initialized${NC}"
else
    echo -e "${YELLOW}⚠️  Configuration will be created on first run${NC}"
fi

echo
echo "========================================"
echo -e "${GREEN}🎉 Installation Complete!${NC}"
echo "========================================"
echo
echo "PlexiChat Client has been installed successfully!"
echo
echo "Installation Details:"
echo "  📁 Install Directory: $INSTALL_DIR"
echo "  ⚙️  Config Directory:  $CONFIG_DIR"
echo "  🖥️  Applications:      $APPS_DIR/PlexiChat.app"
echo "  📱 CLI App:           $APPS_DIR/PlexiChat CLI.app"
echo "  🛤️  PATH:             Updated"
echo
echo "Quick Start:"
echo "  1. Open Terminal"
echo "  2. Run: plexichat-cli config set url \"http://your-server:8000\""
echo "  3. Run: plexichat-cli chat"
echo "  4. Or open PlexiChat.app from Applications"
echo
echo "📚 Documentation: https://github.com/linux-of-user/plexichat-app"
echo "🐛 Issues: https://github.com/linux-of-user/plexichat-app/issues"
echo
echo -e "${GREEN}🚀 Ready to use PlexiChat!${NC}"
echo
