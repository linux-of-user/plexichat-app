# 🤝 Contributing to PlexiChat Desktop

Thank you for your interest in contributing to PlexiChat Desktop! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites
- **Go 1.21+** for backend development
- **CGO enabled** for GUI builds
- **Git** for version control
- **Fyne** for GUI development (`go install fyne.io/fyne/v2/cmd/fyne@latest`)

### Development Setup
```bash
# Clone the repository
git clone https://github.com/yourusername/plexichat-desktop.git
cd plexichat-desktop

# Install dependencies
go mod download

# Build and test
go build -o plexichat-cli.exe .
./plexichat-cli.exe gui
```

## 🎯 How to Contribute

### 🐛 Reporting Bugs
1. **Search existing issues** to avoid duplicates
2. **Use the bug report template** with:
   - Clear description of the issue
   - Steps to reproduce
   - Expected vs actual behavior
   - Screenshots/logs if applicable
   - System information (OS, Go version, etc.)

### 💡 Suggesting Features
1. **Check the roadmap** in issues/projects
2. **Open a feature request** with:
   - Clear use case and motivation
   - Detailed description of proposed solution
   - Alternative solutions considered
   - Mockups/wireframes if applicable

### 🔧 Code Contributions

#### Branch Naming
- `feature/description` - New features
- `bugfix/description` - Bug fixes
- `hotfix/description` - Critical fixes
- `docs/description` - Documentation updates

#### Commit Messages
Follow conventional commits:
```
type(scope): description

feat(gui): add dark theme support
fix(auth): resolve login timeout issue
docs(readme): update installation instructions
```

#### Pull Request Process
1. **Fork the repository** and create a feature branch
2. **Make your changes** with proper testing
3. **Update documentation** if needed
4. **Run tests** and ensure they pass
5. **Submit a pull request** with:
   - Clear title and description
   - Reference to related issues
   - Screenshots for UI changes
   - Testing instructions

## 🏗️ Development Guidelines

### Code Style
- **Go formatting**: Use `gofmt` and `golint`
- **Error handling**: Always handle errors appropriately
- **Comments**: Document public functions and complex logic
- **Testing**: Write tests for new functionality

### GUI Development
- **Fyne best practices**: Follow Fyne design patterns
- **Responsive design**: Ensure UI works at different sizes
- **Accessibility**: Consider keyboard navigation and screen readers
- **Theme support**: Respect system theme preferences

### API Integration
- **Error handling**: Provide meaningful error messages
- **Timeouts**: Use appropriate context timeouts
- **Retry logic**: Implement exponential backoff where appropriate
- **Logging**: Add debug logging for troubleshooting

## 🧪 Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/client
```

### GUI Testing
```bash
# Build and test GUI
fyne package -name PlexiChat-Test
./PlexiChat-Test.exe

# Test CLI with GUI
go build -o test-cli.exe .
./test-cli.exe gui
```

### Manual Testing Checklist
- [ ] Application launches without errors
- [ ] Login flow works with valid/invalid credentials
- [ ] Message sending and receiving functions
- [ ] File upload/download works
- [ ] Settings persist between sessions
- [ ] Error handling displays helpful messages
- [ ] Keyboard shortcuts work as expected

## 📚 Documentation

### Code Documentation
- **Public functions**: Must have Go doc comments
- **Complex algorithms**: Explain the approach
- **Configuration**: Document all settings and options
- **Examples**: Provide usage examples where helpful

### User Documentation
- **README**: Keep installation and usage instructions current
- **CHANGELOG**: Document all changes in releases
- **Screenshots**: Update UI screenshots when interface changes
- **Troubleshooting**: Add common issues and solutions

## 🎨 Design Guidelines

### UI/UX Principles
- **Simplicity**: Keep interfaces clean and intuitive
- **Consistency**: Use consistent patterns and styling
- **Feedback**: Provide clear feedback for user actions
- **Performance**: Ensure responsive and smooth interactions

### Visual Design
- **Icons**: Use consistent icon style (preferably from Fyne theme)
- **Colors**: Respect system theme and accessibility guidelines
- **Typography**: Use appropriate font sizes and weights
- **Spacing**: Maintain consistent margins and padding

## 🚀 Release Process

### Version Numbering
- **Major.Minor.Patch-Stage** (e.g., `2.0.0-alpha`)
- **Stages**: `alpha` → `beta` → `rc` → `stable`
- **Special**: `nightly`, `experimental`, `hotfix`

### Release Checklist
- [ ] All tests pass
- [ ] Documentation updated
- [ ] CHANGELOG updated
- [ ] Version numbers bumped
- [ ] Release notes prepared
- [ ] Cross-platform builds tested
- [ ] GitHub release created

## 🆘 Getting Help

### Community Support
- **GitHub Discussions**: Ask questions and share ideas
- **Issues**: Report bugs and request features
- **Discord/Slack**: Join community chat (if available)

### Development Help
- **Code Review**: Request reviews for complex changes
- **Architecture**: Discuss major changes before implementation
- **Mentoring**: New contributors welcome - ask for guidance!

## 📄 License

By contributing to PlexiChat Desktop, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for helping make PlexiChat Desktop better!** 🚀
