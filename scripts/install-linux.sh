#!/bin/bash

echo "========================================"
echo "PlexiChat Client - Linux Installer"
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
if [[ ! -f "plexichat-cli-linux-amd64" ]] || [[ ! -f "plexichat-gui-linux-amd64" ]]; then
    echo "Downloading latest release..."
    
    # Get latest release info
    REPO="linux-of-user/plexichat-app"
    API_URL="https://api.github.com/repos/$REPO/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        RELEASE_INFO=$(curl -s "$API_URL")
    elif command -v wget >/dev/null 2>&1; then
        RELEASE_INFO=$(wget -qO- "$API_URL")
    else
        echo -e "${RED}❌ Neither curl nor wget found. Please install one of them.${NC}"
        exit 1
    fi
    
    # Extract download URLs
    CLI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*plexichat-cli-linux-amd64[^"]*"' | cut -d'"' -f4)
    GUI_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*plexichat-gui-linux-amd64[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$CLI_URL" ]]; then
        echo "Downloading CLI..."
        if command -v curl >/dev/null 2>&1; then
            curl -L -o "plexichat-cli-linux-amd64" "$CLI_URL"
        else
            wget -O "plexichat-cli-linux-amd64" "$CLI_URL"
        fi
    fi
    
    if [[ -n "$GUI_URL" ]]; then
        echo "Downloading GUI..."
        if command -v curl >/dev/null 2>&1; then
            curl -L -o "plexichat-gui-linux-amd64" "$GUI_URL"
        else
            wget -O "plexichat-gui-linux-amd64" "$GUI_URL"
        fi
    fi
    
    if [[ ! -f "plexichat-cli-linux-amd64" ]] && [[ ! -f "plexichat-gui-linux-amd64" ]]; then
        echo -e "${RED}❌ Failed to download binaries${NC}"
        echo "Please download them manually from:"
        echo "https://github.com/linux-of-user/plexichat-app/releases/latest"
        exit 1
    fi
fi

# Install binaries
echo "[3/6] Installing binaries..."
if [[ -f "plexichat-cli-linux-amd64" ]]; then
    cp "plexichat-cli-linux-amd64" "$INSTALL_DIR/plexichat-cli"
    chmod +x "$INSTALL_DIR/plexichat-cli"
    echo -e "${GREEN}✅ CLI installed${NC}"
else
    echo -e "${RED}❌ CLI binary not found${NC}"
fi

if [[ -f "plexichat-gui-linux-amd64" ]]; then
    cp "plexichat-gui-linux-amd64" "$INSTALL_DIR/plexichat-gui"
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
        SHELL_RC="$HOME/.bashrc"
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

# Create desktop entries (if GUI environment is available)
echo "[5/6] Creating desktop entries..."
if [[ -n "$DISPLAY" ]] || [[ -n "$WAYLAND_DISPLAY" ]]; then
    DESKTOP_DIR="$HOME/.local/share/applications"
    mkdir -p "$DESKTOP_DIR"
    
    # CLI desktop entry
    cat > "$DESKTOP_DIR/plexichat-cli.desktop" << EOF
[Desktop Entry]
Name=PlexiChat CLI
Comment=PlexiChat Command Line Interface
Exec=gnome-terminal -- $INSTALL_DIR/plexichat-cli
Icon=terminal
Type=Application
Categories=Network;Chat;
Terminal=true
EOF
    
    # GUI desktop entry
    cat > "$DESKTOP_DIR/plexichat.desktop" << EOF
[Desktop Entry]
Name=PlexiChat
Comment=PlexiChat Desktop Application
Exec=$INSTALL_DIR/plexichat-gui
Icon=chat
Type=Application
Categories=Network;Chat;
Terminal=false
EOF
    
    # Update desktop database
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$DESKTOP_DIR" 2>/dev/null
    fi
    
    echo -e "${GREEN}✅ Desktop entries created${NC}"
else
    echo -e "${YELLOW}⚠️  No GUI environment detected - skipping desktop entries${NC}"
fi

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
if [[ -n "$DISPLAY" ]] || [[ -n "$WAYLAND_DISPLAY" ]]; then
    echo "  🖥️  Desktop Entries:   Created"
fi
echo "  🛤️  PATH:             Updated"
echo
echo "Quick Start:"
echo "  1. Open a new terminal"
echo "  2. Run: plexichat-cli config set url \"http://your-server:8000\""
echo "  3. Run: plexichat-cli chat"
echo "  4. Or run: plexichat-gui"
echo
echo "📚 Documentation: https://github.com/linux-of-user/plexichat-app"
echo "🐛 Issues: https://github.com/linux-of-user/plexichat-app/issues"
echo
echo -e "${GREEN}🚀 Ready to use PlexiChat!${NC}"
echo
