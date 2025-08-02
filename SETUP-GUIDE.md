# 🔧 PlexiChat Desktop Setup Guide

## 🚨 **IMPORTANT: CGO Setup Required**

PlexiChat Desktop uses Fyne for the GUI, which requires CGO (C bindings). Here's how to set it up properly:

## 🪟 **Windows Setup**

### **Option 1: TDM-GCC (Recommended)**
```cmd
# Download and install TDM-GCC from:
# https://jmeubank.github.io/tdm-gcc/

# After installation, verify:
gcc --version
go env CGO_ENABLED
```

### **Option 2: Visual Studio Build Tools**
```cmd
# Download Visual Studio Build Tools from:
# https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022

# Install C++ build tools
# Restart command prompt after installation
```

### **Option 3: MinGW-w64**
```cmd
# Install via Chocolatey:
choco install mingw

# Or download from:
# https://www.mingw-w64.org/downloads/
```

## 🐧 **Linux Setup**

### **Ubuntu/Debian**
```bash
sudo apt-get update
sudo apt-get install gcc pkg-config libgl1-mesa-dev xorg-dev
```

### **CentOS/RHEL/Fedora**
```bash
sudo yum install gcc pkgconfig mesa-libGL-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel
```

## 🍎 **macOS Setup**

### **Xcode Command Line Tools**
```bash
xcode-select --install
```

## 🚀 **Build Instructions**

### **Step 1: Verify Environment**
```cmd
# Check Go version (1.21+ required)
go version

# Check CGO is enabled
go env CGO_ENABLED

# Should return "1" - if not, run:
set CGO_ENABLED=1
```

### **Step 2: Install Fyne Tools**
```cmd
go install fyne.io/fyne/v2/cmd/fyne@latest
```

### **Step 3: Build PlexiChat**
```cmd
# Clean build
if exist build rmdir /s /q build
mkdir build

# Build GUI with Fyne
fyne package -name PlexiChat-Desktop -icon icon.png
move PlexiChat-Desktop.exe build\

# Build CLI
go build -ldflags "-X main.version=2.0.0-alpha" -o build\PlexiChat-CLI.exe .
```

### **Step 4: Test Build**
```cmd
# Test CLI
build\PlexiChat-CLI.exe --version

# Test GUI
build\PlexiChat-Desktop.exe
```

## 🔧 **Troubleshooting**

### **Error: "build constraints exclude all Go files"**
**Solution:** CGO is not properly enabled or C compiler is missing.
```cmd
# Windows - Install TDM-GCC or Visual Studio Build Tools
# Linux - Install gcc and development headers
# macOS - Install Xcode command line tools

# Then set:
set CGO_ENABLED=1
```

### **Error: "gcc: command not found"**
**Solution:** C compiler not in PATH.
```cmd
# Windows - Add TDM-GCC to PATH
# Linux - Install gcc package
# macOS - Install Xcode tools
```

### **Error: "cannot find package"**
**Solution:** Missing dependencies.
```cmd
go mod download
go mod tidy
```

### **GUI doesn't launch**
**Solution:** Try these steps:
```cmd
# 1. Check if executable was created
dir build\*.exe

# 2. Run with verbose output
build\PlexiChat-CLI.exe gui

# 3. Check for missing DLLs (Windows)
# Install Visual C++ Redistributable if needed

# 4. Try minimal test
go run test-minimal.go
```

## 🎯 **Alternative Options**

### **If GUI Build Fails**
1. **Use CLI with web interface:**
   ```cmd
   build\PlexiChat-CLI.exe web
   # Opens web interface in browser
   ```

2. **Use remote GUI:**
   ```cmd
   # Connect to PlexiChat server web interface directly
   # http://your-server:8000
   ```

3. **Docker build:**
   ```cmd
   # Use Docker with pre-configured build environment
   docker build -t plexichat-desktop .
   ```

## 📋 **Build Environment Checklist**

- [ ] Go 1.21+ installed
- [ ] CGO_ENABLED=1
- [ ] C compiler installed (gcc/clang/MSVC)
- [ ] Fyne tools installed
- [ ] Dependencies downloaded (`go mod download`)
- [ ] Icon file present (`icon.png`)

## 🚀 **Quick Start (If Everything Works)**

```cmd
# One-command build and test
quick-build.cmd && build\PlexiChat-Desktop.exe
```

## 🆘 **Still Having Issues?**

1. **Check the error logs** - Look for specific error messages
2. **Try the minimal test** - Run `go run test-minimal.go`
3. **Use CLI version** - The CLI always works: `go build -o plexichat.exe .`
4. **Report issues** - Open a GitHub issue with your error logs

---

**Once set up correctly, PlexiChat Desktop builds and runs beautifully!** 🎨✨
