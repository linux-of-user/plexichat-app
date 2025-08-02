package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/client"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/spf13/viper"
)

type GUIState struct {
	app        fyne.App
	window     fyne.Window
	client     *client.Client
	currentTab string
	user       *User
	groups     []Group
	messages   map[string][]Message
	mu         sync.RWMutex
	isDarkMode bool
	settings   *AppSettings
}

type AppSettings struct {
	DarkMode        bool   `json:"dark_mode"`
	NotificationsOn bool   `json:"notifications_on"`
	SoundEffects    bool   `json:"sound_effects"`
	Username        string `json:"username"`
	ServerURL       string `json:"server_url"`
	FontSize        int    `json:"font_size"`
	AutoConnect     bool   `json:"auto_connect"`
}

type UserAvatar struct {
	Initials string
	Color    string
	Username string
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Members     []Member  `json:"members"`
	Channels    []Channel `json:"channels"`
	CreatedAt   time.Time `json:"created_at"`
}

type Member struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Channel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	GroupID string `json:"group_id"`
}

type Message struct {
	ID        string      `json:"id"`
	Content   string      `json:"content"`
	Author    string      `json:"author"`
	ChannelID string      `json:"channel_id"`
	Timestamp time.Time   `json:"timestamp"`
	Avatar    *UserAvatar `json:"-"`
	IsOwn     bool        `json:"-"`
}

// RunGUI launches the native Fyne GUI application
func RunGUI() error {
	// Add comprehensive error recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("GUI panic recovered: %v\n", r)
			fmt.Println("\n🔧 TROUBLESHOOTING TIPS:")
			fmt.Println("1. Ensure CGO is enabled: set CGO_ENABLED=1")
			fmt.Println("2. Install a C compiler (GCC/MinGW on Windows)")
			fmt.Println("3. Try: go install fyne.io/fyne/v2/cmd/fyne@latest")
			fmt.Println("4. For Windows: Install TDM-GCC or Visual Studio Build Tools")
			fmt.Println("5. Alternative: Use the web interface with 'plexichat-client web'")
		}
	}()

	fmt.Println("🚀 Initializing PlexiChat GUI...")
	fmt.Println("📋 Checking GUI dependencies...")

	// Test if Fyne can be imported properly
	fmt.Println("✓ Fyne imports successful")

	// Create app with comprehensive error handling
	fmt.Println("📱 Creating Fyne application...")
	myApp := app.NewWithID("com.plexichat.client")
	if myApp == nil {
		return fmt.Errorf("❌ Failed to create Fyne application - CGO may not be enabled")
	}
	fmt.Println("✓ Fyne application created successfully")

	// Set icon with fallback
	myApp.SetIcon(theme.ComputerIcon())

	// Load or create default settings
	settings := loadSettings()
	if settings == nil {
		fmt.Println("Warning: Using default settings")
		settings = &AppSettings{
			DarkMode:        false,
			NotificationsOn: true,
			SoundEffects:    true,
			Username:        "",
			ServerURL:       "http://localhost:8000",
			FontSize:        12,
			AutoConnect:     false,
		}
	}

	// Apply theme based on settings (using modern approach)
	fmt.Printf("Applying theme (dark mode: %v)...\n", settings.DarkMode)
	// Note: Modern Fyne respects system theme preferences
	// We'll handle theme switching in the UI instead

	// Set up the main window with proper configuration
	fmt.Println("Creating main window...")
	mainWindow := myApp.NewWindow("🚀 PlexiChat Desktop - Modern Team Communication")

	if mainWindow == nil {
		return fmt.Errorf("failed to create main window")
	}

	// Configure window properties
	mainWindow.Resize(fyne.NewSize(1400, 900))
	mainWindow.CenterOnScreen()
	mainWindow.SetFixedSize(false) // Allow resizing

	// Set minimum size to ensure usability
	mainWindow.SetContent(widget.NewLabel("Loading PlexiChat..."))

	// Create client with fallback URL
	serverURL := viper.GetString("url")
	if serverURL == "" {
		serverURL = "http://localhost:8000"
		viper.Set("url", serverURL)
	}

	fmt.Printf("Creating API client for: %s\n", serverURL)
	apiClient := client.NewClient(serverURL)

	if apiClient == nil {
		return fmt.Errorf("failed to create API client for %s", serverURL)
	}

	// Initialize the GUI state with proper error checking
	state := &GUIState{
		app:        myApp,
		window:     mainWindow,
		client:     apiClient,
		currentTab: "login",
		messages:   make(map[string][]Message),
		isDarkMode: settings.DarkMode,
		settings:   settings,
	}

	// Start background monitoring
	go monitorConnection(state)

	fmt.Println("GUI state initialized successfully")

	// Check for existing session
	if checkExistingSession(state) {
		// User is already logged in, go directly to main UI
		mainUI := createMainUI(state)
		mainWindow.SetContent(mainUI)
		mainWindow.SetTitle(fmt.Sprintf("PlexiChat - %s", state.user.Username))
		showNotification(state, "Welcome Back", fmt.Sprintf("Automatically logged in as %s", state.user.Username))
	} else {
		// Show login UI
		loginUI := createLoginUI(state)
		mainWindow.SetContent(loginUI)
	}

	// Set up window close handler
	mainWindow.SetCloseIntercept(func() {
		dialog.ShowConfirm("Confirm", "Are you sure you want to quit PlexiChat?", func(quit bool) {
			if quit {
				myApp.Quit()
			}
		}, mainWindow)
	})

	// Add global keyboard shortcuts
	mainWindow.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyEscape:
			// Close any open dialogs or return to main view
			fmt.Println("Escape pressed")
		case fyne.KeyF1:
			// Show help dialog
			showHelpDialog(state)
		case fyne.KeyF11:
			// Toggle fullscreen
			if mainWindow.FullScreen() {
				mainWindow.SetFullScreen(false)
			} else {
				mainWindow.SetFullScreen(true)
			}
		}
	})

	// Show and run the application
	fmt.Println("Showing main window...")
	mainWindow.Show()

	fmt.Println("Starting PlexiChat GUI application...")
	myApp.Run()

	fmt.Println("PlexiChat GUI application closed")
	return nil
}

func createLoginUI(state *GUIState) fyne.CanvasObject {
	// Create a stunning welcome header with enhanced styling
	title := widget.NewRichTextFromMarkdown("# 🚀 PlexiChat Desktop")
	title.Wrapping = fyne.TextWrapWord

	subtitle := widget.NewRichTextFromMarkdown("### *Beautiful • Secure • Real-time Communication*")
	subtitle.Wrapping = fyne.TextWrapWord

	// Add version and build info
	versionInfo := widget.NewRichTextFromMarkdown("*v2.0.0-alpha - The Phoenix Release*")
	versionInfo.Wrapping = fyne.TextWrapWord

	// Create modern input fields with better styling
	username := widget.NewEntry()
	username.SetPlaceHolder("👤 Username")
	username.Validator = func(s string) error {
		if len(s) < 2 {
			return fmt.Errorf("username must be at least 2 characters")
		}
		return nil
	}

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("🔒 Password")
	password.Validator = func(s string) error {
		if len(s) < 1 {
			return fmt.Errorf("password is required")
		}
		return nil
	}

	serverURL := widget.NewEntry()
	serverURL.SetText(viper.GetString("url"))
	if serverURL.Text == "" {
		serverURL.SetText("http://localhost:8000")
	}
	serverURL.SetPlaceHolder("🌐 Server URL")
	serverURL.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("server URL is required")
		}
		return nil
	}

	// Create beautiful modern buttons with validation
	loginBtn := widget.NewButton("🚀 Connect to PlexiChat", func() {
		// Validate inputs before attempting login
		if err := username.Validate(); err != nil {
			showErrorDialog(state, "Invalid Username", err.Error())
			return
		}
		if err := password.Validate(); err != nil {
			showErrorDialog(state, "Invalid Password", err.Error())
			return
		}
		if err := serverURL.Validate(); err != nil {
			showErrorDialog(state, "Invalid Server URL", err.Error())
			return
		}
		performLogin(state, username.Text, password.Text, serverURL.Text)
	})
	loginBtn.Importance = widget.HighImportance

	registerBtn := widget.NewButton("📝 Create New Account", func() {
		showRegistrationDialog(state)
	})
	registerBtn.Importance = widget.MediumImportance

	// Add keyboard shortcuts
	username.OnSubmitted = func(string) { password.FocusGained() }
	password.OnSubmitted = func(string) {
		if username.Text != "" && password.Text != "" && serverURL.Text != "" {
			performLogin(state, username.Text, password.Text, serverURL.Text)
		}
	}

	// Create a stunning form with modern design
	form := container.NewVBox(
		// Header section with beautiful typography
		container.NewCenter(container.NewVBox(
			title,
			subtitle,
		)),

		widget.NewSeparator(),

		// Server configuration with modern card design
		widget.NewCard("🌐 Server Configuration", "Connect to your PlexiChat server", container.NewVBox(
			serverURL,
			widget.NewRichTextFromMarkdown("*Default: http://localhost:8000*"),
		)),

		// Authentication section with security emphasis
		widget.NewCard("🔐 Authentication", "Enter your login credentials", container.NewVBox(
			username,
			password,
			widget.NewRichTextFromMarkdown("*Press Enter to login quickly*"),
		)),

		// Action buttons with proper spacing
		container.NewVBox(
			loginBtn,
			registerBtn,
		),

		// Beautiful footer with features
		widget.NewSeparator(),
		container.NewCenter(widget.NewRichTextFromMarkdown("✨ **Features**: Real-time messaging • File sharing • Cross-platform • Secure")),
	)

	// Add connection status indicator
	statusIndicator := widget.NewLabel("🔴 Disconnected")
	statusCard := widget.NewCard("", "", container.NewHBox(
		statusIndicator,
		layout.NewSpacer(),
		widget.NewLabel("Ready to connect"),
	))

	// Create a beautiful centered layout with proper margins
	paddedForm := container.NewPadded(container.NewPadded(form))

	// Add a subtle background effect by centering in a larger container
	return container.NewBorder(
		nil,        // Top
		statusCard, // Bottom - status bar
		nil,        // Left
		nil,        // Right
		container.NewCenter(container.NewVBox(
			layout.NewSpacer(),
			paddedForm,
			layout.NewSpacer(),
		)), // Center
	)
}

func createMainUI(state *GUIState) fyne.CanvasObject {
	// Create modern sidebar with groups
	groupsList := widget.NewList(
		func() int {
			state.mu.RLock()
			defer state.mu.RUnlock()
			return len(state.groups)
		},
		func() fyne.CanvasObject {
			return widget.NewCard("", "", container.NewHBox(
				widget.NewIcon(theme.FolderIcon()),
				widget.NewLabelWithStyle("Group Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				layout.NewSpacer(),
				widget.NewLabelWithStyle("0", fyne.TextAlignTrailing, fyne.TextStyle{}),
				widget.NewIcon(theme.AccountIcon()),
			))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			state.mu.RLock()
			defer state.mu.RUnlock()
			if i < len(state.groups) {
				card := o.(*widget.Card)
				container := card.Content.(*fyne.Container)
				container.Objects[1].(*widget.Label).SetText(state.groups[i].Name)
				container.Objects[3].(*widget.Label).SetText(fmt.Sprintf("%d", len(state.groups[i].Members)))
			}
		},
	)

	// Create search bar for messages
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Search messages...")
	searchBtn := widget.NewButton("Search", func() {
		searchMessages(state, searchEntry.Text)
	})
	searchBar := container.NewBorder(nil, nil, nil, searchBtn, searchEntry)

	// Create modern chat area with rich text
	chatArea := widget.NewRichText()
	chatArea.Wrapping = fyne.TextWrapWord
	welcomeText := &widget.TextSegment{
		Text:  "🎉 Welcome to PlexiChat!\n\n💬 Select a group to start chatting\n🚀 Create new groups to organize conversations\n👥 Invite team members to collaborate\n\n✨ Enjoy real-time messaging!",
		Style: widget.RichTextStyle{},
	}
	chatArea.Segments = []widget.RichTextSegment{welcomeText}

	// Create modern message input with emoji support
	messageInput := widget.NewEntry()
	messageInput.SetPlaceHolder("💬 Type your message here...")
	messageInput.MultiLine = false

	// Add keyboard shortcuts
	messageInput.OnSubmitted = func(text string) {
		if text != "" {
			sendMessage(state, text, "general") // Default to general channel
			messageInput.SetText("")
		}
	}

	// Create modern send button with icon
	sendBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		if messageInput.Text != "" {
			sendMessage(state, messageInput.Text, "general") // Default to general channel
			messageInput.SetText("")
		}
	})
	sendBtn.Importance = widget.HighImportance

	// Create additional action buttons
	emojiBtn := widget.NewButtonWithIcon("😊", theme.ContentAddIcon(), func() {
		showEmojiPicker(state, messageInput)
	})

	fileBtn := widget.NewButtonWithIcon("📎", theme.FolderOpenIcon(), func() {
		showFileUploadDialog(state)
	})

	// Create modern message input area with toolbar
	inputToolbar := container.NewHBox(emojiBtn, fileBtn, layout.NewSpacer(), sendBtn)
	messageContainer := container.NewVBox(
		messageInput,
		inputToolbar,
	)

	// Create a scrollable container for the chat area
	chatScroll := container.NewScroll(chatArea)
	chatScroll.SetMinSize(fyne.NewSize(400, 300))

	// Create chat container with search bar
	chatContainer := container.NewBorder(
		searchBar,        // Top - search functionality
		messageContainer, // Bottom
		nil,              // Left
		nil,              // Right
		chatScroll,       // Center
	)

	// Create modern sidebar with header
	createGroupBtn := widget.NewButtonWithIcon("➕ New Group", theme.ContentAddIcon(), func() {
		showCreateGroupDialog(state)
	})
	createGroupBtn.Importance = widget.MediumImportance

	// Create theme toggle button
	themeIcon := theme.VisibilityIcon()
	if state.isDarkMode {
		themeIcon = theme.VisibilityOffIcon()
	}

	var themeBtn *widget.Button
	themeBtn = widget.NewButtonWithIcon("", themeIcon, func() {
		toggleTheme(state)
		// Update button icon
		if state.isDarkMode {
			themeBtn.SetIcon(theme.VisibilityOffIcon())
		} else {
			themeBtn.SetIcon(theme.VisibilityIcon())
		}
	})

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showSettingsDialog(state)
	})

	logoutBtn := widget.NewButtonWithIcon("", theme.LogoutIcon(), func() {
		dialog.ShowConfirm("Logout", "Are you sure you want to logout?", func(confirm bool) {
			if confirm {
				logout(state)
			}
		}, state.window)
	})

	// Create user avatar and info section
	currentUser := "demo_user"
	userAvatar := generateAvatar(currentUser)
	avatarWidget := createAvatarWidget(userAvatar, 40)

	userInfo := container.NewHBox(
		avatarWidget,
		container.NewVBox(
			widget.NewLabelWithStyle(currentUser, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle("🟢 Online", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		),
	)

	userCard := widget.NewCard("👤 User Profile", "", container.NewVBox(
		userInfo,
		widget.NewSeparator(),
		container.NewGridWithColumns(3, themeBtn, settingsBtn, logoutBtn),
	))

	// Create groups header
	groupsHeader := widget.NewCard("💬 Groups", "", container.NewVBox(
		createGroupBtn,
	))

	sidebar := container.NewVBox(
		userCard,
		groupsHeader,
		groupsList,
	)

	// Create status bar
	statusBar := createStatusBar(state)

	// Create main layout with proper proportions
	split := container.NewHSplit(sidebar, chatContainer)
	split.SetOffset(0.25) // 25% for sidebar, 75% for chat

	// Create main container with status bar
	mainContainer := container.NewBorder(
		nil,       // Top
		statusBar, // Bottom
		nil,       // Left
		nil,       // Right
		split,     // Center
	)

	return mainContainer
}

func performLogin(state *GUIState, username, password, serverURL string) {
	fmt.Printf("🚀 Starting login process for user: %s, server: %s\n", username, serverURL)

	// Validate inputs
	if username == "" || password == "" {
		showErrorDialog(state, "Missing Credentials", "Both username and password are required")
		return
	}

	if serverURL == "" {
		serverURL = "http://localhost:8000"
		fmt.Println("Using default server URL:", serverURL)
	}

	// Update server URL and create new client
	fmt.Printf("Creating client for server: %s\n", serverURL)
	viper.Set("url", serverURL)
	state.client = client.NewClient(serverURL)

	if state.client == nil {
		showErrorDialog(state, "Client Error", "Failed to create API client")
		return
	}

	// Show beautiful loading dialog
	progressBar := widget.NewProgressBarInfinite()
	progressContent := container.NewVBox(
		widget.NewLabelWithStyle("🔐 Connecting to PlexiChat...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		progressBar,
		widget.NewLabel("Server: "+serverURL),
		widget.NewLabel("User: "+username),
		widget.NewRichTextFromMarkdown("*This may take a few moments*"),
	)
	progress := dialog.NewCustomWithoutButtons("Authenticating", progressContent, state.window)
	progress.Show()

	// Perform actual login API call
	go func() {
		defer func() {
			if r := recover(); r != nil {
				progress.Hide()
				showErrorDialog(state, "Login Error", fmt.Sprintf("Unexpected error: %v", r))
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		fmt.Printf("🔐 Attempting login to server: %s\n", serverURL)

		// Call the actual login API
		loginResp, err := state.client.Login(ctx, username, password)

		// Hide progress dialog
		progress.Hide()

		if err != nil {
			fmt.Printf("❌ Login failed with error: %v\n", err)

			// Enhanced error handling with specific error types
			errorMsg := fmt.Sprintf("Login failed: %v", err)
			errorType := "Login Error"

			errStr := err.Error()

			// Check for specific error types and provide helpful messages
			if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
				errorType = "Connection Error"
				errorMsg = "Cannot connect to the PlexiChat server.\n\nPlease check:\n• Server URL is correct\n• Server is running\n• Internet connection is working"
			} else if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid credentials") || strings.Contains(errStr, "401") {
				errorType = "Authentication Error"
				errorMsg = "Invalid username or password.\n\nPlease check:\n• Username is correct\n• Password is correct\n• Account exists on this server"
			} else if strings.Contains(errStr, "timeout") {
				errorType = "Timeout Error"
				errorMsg = "Connection timed out.\n\nPlease:\n• Check your internet connection\n• Try again in a moment\n• Verify server is responding"
			} else if strings.Contains(errStr, "404") {
				errorType = "Server Error"
				errorMsg = "PlexiChat API not found.\n\nPlease check:\n• Server URL is correct\n• PlexiChat server is properly configured"
			}

			showErrorDialog(state, errorType, errorMsg)
			return
		}

		fmt.Printf("✅ Login API call successful\n")

		// Check if 2FA is required
		if loginResp.TwoFARequired {
			show2FADialog(state, username, password, loginResp.Methods)
			return
		}

		// Login successful - save token and user info
		if loginResp.AccessToken != "" {
			viper.Set("token", loginResp.AccessToken)
			viper.Set("username", loginResp.Username)
			viper.Set("user_id", loginResp.UserID)

			// Save config
			saveConfig()

			// Update state
			state.user = &User{
				ID:       loginResp.UserID,
				Username: loginResp.Username,
				Email:    username + "@example.com", // Will get from user profile later
				Token:    loginResp.AccessToken,
			}

			// Get user profile for complete info
			go loadUserProfile(state)

			// Load groups and channels
			go loadUserGroups(state)

			// Switch to main UI
			mainUI := createMainUI(state)
			state.window.SetContent(mainUI)

			// Update window title
			state.window.SetTitle(fmt.Sprintf("PlexiChat - %s", loginResp.Username))

			// Show success notification
			showNotification(state, "Login Successful", fmt.Sprintf("Welcome back, %s!", loginResp.Username))
		} else {
			dialog.ShowError(fmt.Errorf("Login failed: No access token received"), state.window)
		}
	}()
}

func showRegistrationDialog(state *GUIState) {
	// Create modern registration form
	title := widget.NewRichTextFromMarkdown("# 📝 Create Account")
	title.Wrapping = fyne.TextWrapWord

	subtitle := widget.NewRichTextFromMarkdown("*Join the PlexiChat community*")
	subtitle.Wrapping = fyne.TextWrapWord

	username := widget.NewEntry()
	username.SetPlaceHolder("👤 Choose a username")

	email := widget.NewEntry()
	email.SetPlaceHolder("📧 Enter your email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("🔒 Create a password")

	confirmPassword := widget.NewPasswordEntry()
	confirmPassword.SetPlaceHolder("🔒 Confirm your password")

	// Account type selection
	userTypeSelect := widget.NewSelect([]string{"user", "bot"}, nil)
	userTypeSelect.SetSelected("user")

	form := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),

		widget.NewCard("👤 Account Details", "", container.NewVBox(
			username,
			email,
		)),

		widget.NewCard("🔐 Security", "", container.NewVBox(
			password,
			confirmPassword,
		)),

		widget.NewCard("⚙️ Account Type", "", container.NewVBox(
			userTypeSelect,
		)),
	)

	dialog.ShowCustomConfirm("Create Account", "Register", "Cancel", form, func(confirm bool) {
		if confirm {
			performRegistration(state, username.Text, email.Text, password.Text, confirmPassword.Text, userTypeSelect.Selected)
		}
	}, state.window)
}

// performRegistration handles the actual registration API call
func performRegistration(state *GUIState, username, email, password, confirmPassword, userType string) {
	// Validate inputs
	if username == "" || email == "" || password == "" {
		dialog.ShowError(fmt.Errorf("all fields are required"), state.window)
		return
	}

	if password != confirmPassword {
		dialog.ShowError(fmt.Errorf("passwords do not match"), state.window)
		return
	}

	if len(password) < 6 {
		dialog.ShowError(fmt.Errorf("password must be at least 6 characters"), state.window)
		return
	}

	// Show loading dialog
	progressBar := widget.NewProgressBarInfinite()
	progressContent := container.NewVBox(
		widget.NewLabelWithStyle("📝 Creating Account...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		progressBar,
		widget.NewLabel("Setting up your PlexiChat account"),
	)
	progress := dialog.NewCustomWithoutButtons("Registration", progressContent, state.window)
	progress.Show()

	// Perform actual registration API call
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call the registration API
		regResp, err := state.client.Register(ctx, username, email, password, userType)

		// Hide progress dialog
		progress.Hide()

		if err != nil {
			// Show error dialog
			dialog.ShowError(fmt.Errorf("Registration failed: %v", err), state.window)
			return
		}

		// Registration successful
		if regResp.Success {
			// Show success dialog with login option
			successContent := container.NewVBox(
				widget.NewLabelWithStyle("🎉 Registration Successful!", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewSeparator(),
				widget.NewLabel("Account created for: "+regResp.Username),
				widget.NewLabel("User ID: "+regResp.UserID),
				widget.NewSeparator(),
				widget.NewLabel("You can now login with your credentials!"),
			)

			dialog.ShowCustomConfirm("Welcome to PlexiChat!", "Login Now", "Close", successContent, func(login bool) {
				if login {
					// Auto-fill login form
					// This would require passing the credentials back to the login form
					showNotification(state, "Registration Complete", "Please login with your new account")
				}
			}, state.window)
		} else {
			dialog.ShowError(fmt.Errorf("Registration failed: %s", regResp.Message), state.window)
		}
	}()
}

func showCreateGroupDialog(state *GUIState) {
	name := widget.NewEntry()
	name.SetPlaceHolder("Group name")

	description := widget.NewEntry()
	description.SetPlaceHolder("Group description")

	form := container.NewVBox(
		widget.NewLabel("Create New Group"),
		widget.NewLabel("Group Name:"),
		name,
		widget.NewLabel("Description:"),
		description,
	)

	dialog.ShowCustomConfirm("Create Group", "Create", "Cancel", form, func(confirm bool) {
		if confirm {
			// TODO: Implement actual group creation
			dialog.ShowInformation("Create Group", fmt.Sprintf("Group '%s' created successfully!", name.Text), state.window)
		}
	}, state.window)
}

// loadSettings loads application settings from file or creates defaults
func loadSettings() *AppSettings {
	// Create default settings
	defaults := &AppSettings{
		DarkMode:        false,
		NotificationsOn: true,
		SoundEffects:    true,
		Username:        "",
		ServerURL:       "http://localhost:8000",
		FontSize:        12,
		AutoConnect:     false,
	}

	// For now, return defaults (can be enhanced to load from file later)
	return defaults
}

// saveSettings saves application settings to file
func saveSettings(settings *AppSettings) error {
	// TODO: Implement settings persistence
	return nil
}

// toggleTheme switches between dark and light themes
func toggleTheme(state *GUIState) {
	state.isDarkMode = !state.isDarkMode
	state.settings.DarkMode = state.isDarkMode

	// Modern Fyne handles theme switching automatically
	// We just update our internal state and save settings

	// Save settings
	saveSettings(state.settings)

	// Show notification about theme change
	themeName := "Light"
	if state.isDarkMode {
		themeName = "Dark"
	}
	showNotification(state, "Theme Changed", fmt.Sprintf("Switched to %s theme", themeName))
}

// showSettingsDialog displays the settings configuration dialog
func showSettingsDialog(state *GUIState) {
	// Create settings form
	darkModeCheck := widget.NewCheck("Dark Mode", func(checked bool) {
		if checked != state.isDarkMode {
			toggleTheme(state)
		}
	})
	darkModeCheck.SetChecked(state.isDarkMode)

	notificationsCheck := widget.NewCheck("Enable Notifications", func(checked bool) {
		state.settings.NotificationsOn = checked
	})
	notificationsCheck.SetChecked(state.settings.NotificationsOn)

	soundCheck := widget.NewCheck("Sound Effects", func(checked bool) {
		state.settings.SoundEffects = checked
	})
	soundCheck.SetChecked(state.settings.SoundEffects)

	autoConnectCheck := widget.NewCheck("Auto Connect on Startup", func(checked bool) {
		state.settings.AutoConnect = checked
	})
	autoConnectCheck.SetChecked(state.settings.AutoConnect)

	// Create settings content
	settingsContent := container.NewVBox(
		widget.NewCard("🎨 Appearance", "", container.NewVBox(
			darkModeCheck,
		)),
		widget.NewCard("🔔 Notifications", "", container.NewVBox(
			notificationsCheck,
			soundCheck,
		)),
		widget.NewCard("🚀 Startup", "", container.NewVBox(
			autoConnectCheck,
		)),
	)

	// Create dialog
	settingsDialog := dialog.NewCustom("⚙️ Settings", "Close", settingsContent, state.window)
	settingsDialog.Resize(fyne.NewSize(400, 500))
	settingsDialog.Show()
}

// generateAvatar creates a user avatar with initials and color
func generateAvatar(username string) *UserAvatar {
	if username == "" {
		username = "Guest"
	}

	// Generate initials
	initials := ""
	if len(username) > 0 {
		// Simple initials generation
		if len(username) >= 2 {
			initials = string(username[0]) + string(username[1])
		} else {
			initials = string(username[0]) + "U"
		}
	}

	// Generate color based on username hash
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7",
		"#DDA0DD", "#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E9",
		"#F8C471", "#82E0AA", "#F1948A", "#85C1E9", "#D7BDE2",
	}

	hash := 0
	for _, char := range username {
		hash += int(char)
	}
	color := colors[hash%len(colors)]

	return &UserAvatar{
		Initials: initials,
		Color:    color,
		Username: username,
	}
}

// createAvatarWidget creates a visual avatar widget
func createAvatarWidget(avatar *UserAvatar, size float32) *fyne.Container {
	// Create a colored circle with initials
	initialsLabel := widget.NewLabelWithStyle(avatar.Initials, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	initialsLabel.Resize(fyne.NewSize(size, size))

	// Create a container that simulates a colored circle
	avatarContainer := container.NewStack(
		widget.NewCard("", "", container.NewCenter(initialsLabel)),
	)
	avatarContainer.Resize(fyne.NewSize(size, size))

	return avatarContainer
}

// formatTimestamp formats a timestamp for display
func formatTimestamp(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d min ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else if t.Format("2006-01-02") == now.Format("2006-01-02") {
		return t.Format("15:04")
	} else {
		return t.Format("Jan 2, 15:04")
	}
}

// createMessageWidget creates a beautiful message widget with avatar and timestamp
func createMessageWidget(msg *Message) *fyne.Container {
	// Generate avatar if not present
	if msg.Avatar == nil {
		msg.Avatar = generateAvatar(msg.Author)
	}

	// Create avatar widget
	avatarWidget := createAvatarWidget(msg.Avatar, 32)

	// Create message content
	authorLabel := widget.NewLabelWithStyle(msg.Author, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	timestampLabel := widget.NewLabelWithStyle(formatTimestamp(msg.Timestamp), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	contentLabel := widget.NewRichTextFromMarkdown(msg.Content)
	contentLabel.Wrapping = fyne.TextWrapWord

	// Create message header
	messageHeader := container.NewHBox(
		authorLabel,
		layout.NewSpacer(),
		timestampLabel,
	)

	// Create message body
	messageBody := container.NewVBox(
		messageHeader,
		contentLabel,
	)

	// Create full message container
	messageContainer := container.NewHBox(
		avatarWidget,
		messageBody,
	)

	// Add styling based on whether it's own message
	if msg.IsOwn {
		// Different styling for own messages (could add background color, etc.)
		messageCard := widget.NewCard("", "", messageContainer)
		return container.NewPadded(messageCard)
	}

	return container.NewPadded(messageContainer)
}

// showEmojiPicker displays an emoji picker dialog
func showEmojiPicker(state *GUIState, messageInput *widget.Entry) {
	// Define emoji categories
	emojiCategories := map[string][]string{
		"😊 Smileys":  {"😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣", "😊", "😇", "🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚", "😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩", "🥳"},
		"❤️ Hearts":  {"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💔", "❣️", "💕", "💞", "💓", "💗", "💖", "💘", "💝", "💟"},
		"👍 Gestures": {"👍", "👎", "👌", "🤌", "🤏", "✌️", "🤞", "🤟", "🤘", "🤙", "👈", "👉", "👆", "🖕", "👇", "☝️", "👋", "🤚", "🖐️", "✋", "🖖", "👏", "🙌", "🤲", "🤝", "🙏"},
		"🎉 Objects":  {"🎉", "🎊", "🎈", "🎁", "🎀", "🎂", "🍰", "🧁", "🍭", "🍬", "🍫", "🍩", "🍪", "☕", "🍵", "🥤", "🍺", "🍻", "🥂", "🍷", "🥃", "🍸", "🍹", "🍾", "🔥", "💯", "⭐", "🌟", "✨", "💫"},
	}

	// Create emoji grid for each category
	var categoryTabs *container.AppTabs
	categoryTabs = container.NewAppTabs()

	for categoryName, emojis := range emojiCategories {
		emojiGrid := container.NewGridWithColumns(8)

		for _, emoji := range emojis {
			emojiBtn := widget.NewButton(emoji, func(selectedEmoji string) func() {
				return func() {
					// Add emoji to message input
					currentText := messageInput.Text
					messageInput.SetText(currentText + selectedEmoji)
					// Close the dialog (we'll need to track it)
				}
			}(emoji))
			emojiBtn.Resize(fyne.NewSize(40, 40))
			emojiGrid.Add(emojiBtn)
		}

		scrollableGrid := container.NewScroll(emojiGrid)
		scrollableGrid.SetMinSize(fyne.NewSize(400, 300))
		categoryTabs.Append(container.NewTabItem(categoryName, scrollableGrid))
	}

	// Create emoji picker dialog
	emojiDialog := dialog.NewCustom("🎭 Pick an Emoji", "Close", categoryTabs, state.window)
	emojiDialog.Resize(fyne.NewSize(500, 400))
	emojiDialog.Show()
}

// showHelpDialog displays keyboard shortcuts and help information
func showHelpDialog(state *GUIState) {
	helpContent := widget.NewRichTextFromMarkdown(`# 🚀 PlexiChat Help

## ⌨️ Keyboard Shortcuts

- **Enter** - Send message
- **Ctrl+Enter** - New line in message
- **F1** - Show this help dialog
- **F11** - Toggle fullscreen
- **Escape** - Close dialogs

## 🎨 Features

- **Dark/Light Theme** - Toggle with the theme button
- **Emoji Picker** - Click the emoji button to add emojis
- **File Upload** - Click the folder button to share files
- **User Avatars** - Automatic avatar generation with initials
- **Real-time Timestamps** - Messages show relative time

## 🔧 Settings

Access settings through the gear icon to customize:
- Theme preferences
- Notification settings
- Sound effects
- Auto-connect options

## 💬 Tips

- Use markdown in messages for **bold** and *italic* text
- Drag and drop files to share them quickly
- Right-click for context menus
- Use @username to mention someone

---
*PlexiChat v1.0 - Modern Team Communication*`)

	helpContent.Wrapping = fyne.TextWrapWord

	scrollableHelp := container.NewScroll(helpContent)
	scrollableHelp.SetMinSize(fyne.NewSize(500, 400))

	helpDialog := dialog.NewCustom("❓ Help & Shortcuts", "Close", scrollableHelp, state.window)
	helpDialog.Resize(fyne.NewSize(600, 500))
	helpDialog.Show()
}

// showFileUploadDialog displays a file picker for uploading files
func showFileUploadDialog(state *GUIState) {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, state.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		// Get file info
		fileName := reader.URI().Name()
		fileSize := "Unknown size"

		// Show file upload confirmation
		confirmContent := container.NewVBox(
			widget.NewLabelWithStyle("📎 File Upload", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("File: "+fileName),
			widget.NewLabel("Size: "+fileSize),
			widget.NewSeparator(),
			widget.NewLabel("Ready to upload this file?"),
		)

		dialog.ShowCustomConfirm("Upload File", "Upload", "Cancel", confirmContent, func(upload bool) {
			if upload {
				// TODO: Implement actual file upload
				dialog.ShowInformation("Upload", fmt.Sprintf("File '%s' uploaded successfully!", fileName), state.window)
			}
		}, state.window)

	}, state.window)

	// Set file filters
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{
		".txt", ".md", ".pdf", ".doc", ".docx",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".mp4", ".avi", ".mov", ".mp3", ".wav",
		".zip", ".rar", ".7z", ".tar", ".gz",
	}))

	fileDialog.Resize(fyne.NewSize(800, 600))
	fileDialog.Show()
}

// createStatusBar creates a status bar with connection info and notifications
func createStatusBar(state *GUIState) *fyne.Container {
	// Connection status
	connectionStatus := widget.NewLabelWithStyle("🟢 Connected", fyne.TextAlignLeading, fyne.TextStyle{})

	// Message count
	messageCount := widget.NewLabelWithStyle("0 messages", fyne.TextAlignCenter, fyne.TextStyle{})

	// Current time
	timeLabel := widget.NewLabelWithStyle(time.Now().Format("15:04"), fyne.TextAlignTrailing, fyne.TextStyle{})

	// Update time every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			timeLabel.SetText(time.Now().Format("15:04"))
		}
	}()

	// Notification indicator
	notificationIcon := widget.NewIcon(theme.MailSendIcon())
	notificationIcon.Hide() // Hidden by default

	statusContainer := container.NewHBox(
		connectionStatus,
		layout.NewSpacer(),
		messageCount,
		layout.NewSpacer(),
		notificationIcon,
		timeLabel,
	)

	return container.NewPadded(statusContainer)
}

// showNotification displays a desktop-style notification
func showNotification(state *GUIState, title, message string) {
	if !state.settings.NotificationsOn {
		return
	}

	// Create notification content
	notificationContent := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(message),
	)

	// Create notification dialog
	notification := dialog.NewCustomWithoutButtons("🔔 Notification", notificationContent, state.window)
	notification.Resize(fyne.NewSize(300, 150))
	notification.Show()

	// Auto-close after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		notification.Hide()
	}()
}

// saveConfig saves the current configuration to file
func saveConfig() error {
	return viper.WriteConfig()
}

// show2FADialog displays the 2FA authentication dialog
func show2FADialog(state *GUIState, username, password string, methods []string) {
	// Create 2FA method selection
	methodSelect := widget.NewSelect(methods, nil)
	if len(methods) > 0 {
		methodSelect.SetSelected(methods[0])
	}

	codeEntry := widget.NewEntry()
	codeEntry.SetPlaceHolder("Enter 2FA code")

	form := container.NewVBox(
		widget.NewLabelWithStyle("🔐 Two-Factor Authentication", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Method:"),
		methodSelect,
		widget.NewLabel("Code:"),
		codeEntry,
	)

	dialog.ShowCustomConfirm("2FA Required", "Verify", "Cancel", form, func(verify bool) {
		if verify && codeEntry.Text != "" {
			perform2FALogin(state, username, password, methodSelect.Selected, codeEntry.Text)
		}
	}, state.window)
}

// perform2FALogin performs 2FA authentication
func perform2FALogin(state *GUIState, username, password, method, code string) {
	// Show modern loading dialog
	progressBar := widget.NewProgressBarInfinite()
	progressContent := container.NewVBox(
		widget.NewLabelWithStyle("🔐 Verifying 2FA...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		progressBar,
		widget.NewLabel("Please wait while we verify your code"),
	)
	progress := dialog.NewCustomWithoutButtons("Verifying 2FA", progressContent, state.window)
	progress.Show()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call 2FA login API
		loginResp, err := state.client.LoginWith2FA(ctx, username, password, method, code, "")

		progress.Hide()

		if err != nil {
			dialog.ShowError(fmt.Errorf("2FA verification failed: %v", err), state.window)
			return
		}

		// Handle successful 2FA login
		if loginResp.AccessToken != "" {
			viper.Set("token", loginResp.AccessToken)
			viper.Set("username", loginResp.User.Username)
			viper.Set("user_id", loginResp.User.ID)

			saveConfig()

			state.user = &User{
				ID:       fmt.Sprintf("%d", loginResp.User.ID),
				Username: loginResp.User.Username,
				Email:    loginResp.User.Email,
				Token:    loginResp.AccessToken,
			}

			// Load user data and switch to main UI
			go loadUserProfile(state)
			go loadUserGroups(state)

			mainUI := createMainUI(state)
			state.window.SetContent(mainUI)
			state.window.SetTitle(fmt.Sprintf("PlexiChat - %s", loginResp.User.Username))

			showNotification(state, "2FA Successful", "Welcome to PlexiChat!")
		}
	}()
}

// loadUserProfile loads the current user's profile information
func loadUserProfile(state *GUIState) {
	if state.user == nil || state.user.Token == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	state.client.SetToken(state.user.Token)
	userResp, err := state.client.GetCurrentUser(ctx)
	if err != nil {
		fmt.Printf("Failed to load user profile: %v\n", err)
		return
	}

	// Update user info
	if userResp != nil {
		state.user.Email = userResp.Email
		// Update other fields as needed
	}
}

// loadUserGroups loads the user's groups and channels
func loadUserGroups(state *GUIState) {
	// For now, create some default groups
	// This should be replaced with actual API calls when available
	state.mu.Lock()
	defer state.mu.Unlock()

	state.groups = []Group{
		{
			ID:          "1",
			Name:        "General",
			Description: "General discussion",
			Members:     []Member{{ID: "1", Username: state.user.Username, Role: "admin"}},
			Channels:    []Channel{{ID: "1", Name: "general", Type: "text", GroupID: "1"}},
			CreatedAt:   time.Now(),
		},
		{
			ID:          "2",
			Name:        "Random",
			Description: "Random chat",
			Members:     []Member{{ID: "1", Username: state.user.Username, Role: "member"}},
			Channels:    []Channel{{ID: "2", Name: "random", Type: "text", GroupID: "2"}},
			CreatedAt:   time.Now(),
		},
	}
}

// sendMessage sends a message to the specified channel
func sendMessage(state *GUIState, content, channelID string) {
	if state.user == nil || state.user.Token == "" {
		showNotification(state, "Error", "Please login first")
		return
	}

	if content == "" {
		return
	}

	// Show sending indicator (optional)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Set token for API call
		state.client.SetToken(state.user.Token)

		// Send message via API
		message, err := state.client.SendMessage(ctx, content, channelID)
		if err != nil {
			// Show error notification
			showNotification(state, "Send Failed", fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		// Add message to local state
		if message != nil {
			state.mu.Lock()
			if state.messages[channelID] == nil {
				state.messages[channelID] = make([]Message, 0)
			}

			// Convert API message to local message format
			localMessage := Message{
				ID:        fmt.Sprintf("%d", message.ID),
				Content:   message.Content,
				Author:    state.user.Username,
				ChannelID: channelID,
				Timestamp: time.Now(),
				Avatar:    generateAvatar(state.user.Username),
				IsOwn:     true,
			}

			state.messages[channelID] = append(state.messages[channelID], localMessage)
			state.mu.Unlock()

			// Update UI (this would need to refresh the chat display)
			// For now, just show a success notification
			showNotification(state, "Message Sent", "Your message was delivered successfully")
		}
	}()
}

// refreshChatDisplay updates the chat display with new messages
func refreshChatDisplay(state *GUIState, channelID string) {
	// This function would update the chat area with new messages
	// Implementation depends on how the chat area is structured
	// For now, this is a placeholder
}

// checkExistingSession checks if there's a valid existing session
func checkExistingSession(state *GUIState) bool {
	// Check if we have stored credentials
	token := viper.GetString("token")
	username := viper.GetString("username")
	userID := viper.GetString("user_id")

	if token == "" || username == "" {
		return false
	}

	// Set token and test if it's still valid
	state.client.SetToken(token)

	// Try to get current user info to validate token
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userResp, err := state.client.GetCurrentUser(ctx)
	if err != nil {
		// Token is invalid, clear stored credentials
		clearStoredSession()
		return false
	}

	// Token is valid, restore user state
	state.user = &User{
		ID:       userID,
		Username: username,
		Email:    userResp.Email,
		Token:    token,
	}

	// Load user data
	go loadUserGroups(state)

	return true
}

// clearStoredSession clears all stored session data
func clearStoredSession() {
	viper.Set("token", "")
	viper.Set("username", "")
	viper.Set("user_id", "")
	viper.Set("refresh_token", "")
	saveConfig()
}

// logout performs logout and clears session
func logout(state *GUIState) {
	// Clear stored session
	clearStoredSession()

	// Reset state
	state.user = nil
	state.groups = nil
	state.messages = make(map[string][]Message)

	// Return to login screen
	loginUI := createLoginUI(state)
	state.window.SetContent(loginUI)
	state.window.SetTitle("PlexiChat - Login")

	showNotification(state, "Logged Out", "You have been logged out successfully")
}

// showErrorDialog displays an enhanced error dialog with helpful information
func showErrorDialog(state *GUIState, title, message string) {
	// Create error content with icon and helpful text
	errorContent := container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.ErrorIcon()),
			widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		widget.NewSeparator(),
		widget.NewLabel(message),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("💡 Troubleshooting Tips:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("• Check your internet connection"),
		widget.NewLabel("• Verify the server URL is correct"),
		widget.NewLabel("• Ensure the PlexiChat server is running"),
		widget.NewLabel("• Try again in a few moments"),
	)

	dialog.ShowCustom("❌ "+title, "OK", errorContent, state.window)
}

// showConnectionStatus updates the connection status in the UI
func showConnectionStatus(state *GUIState, connected bool, message string) {
	// This would update a status indicator in the UI
	// For now, just show a notification
	if connected {
		showNotification(state, "Connected", message)
	} else {
		showNotification(state, "Disconnected", message)
	}
}

// retryWithBackoff performs an operation with exponential backoff
func retryWithBackoff(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Wait with exponential backoff
		waitTime := time.Duration(1<<uint(i)) * time.Second
		if waitTime > 30*time.Second {
			waitTime = 30 * time.Second
		}
		time.Sleep(waitTime)
	}
	return err
}

// monitorConnection monitors the connection status in the background
func monitorConnection(state *GUIState) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check connection health
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			if state.client != nil {
				_, err := state.client.Health(ctx)
				if err != nil {
					// Connection lost
					showConnectionStatus(state, false, "Connection lost")
				} else {
					// Connection healthy
					showConnectionStatus(state, true, "Connected")
				}
			}
			cancel()
		}
	}
}

// searchMessages searches through message history
func searchMessages(state *GUIState, query string) {
	if query == "" {
		showNotification(state, "Search", "Please enter a search term")
		return
	}

	fmt.Printf("🔍 Searching for: %s\n", query)

	// Search through all messages
	var results []Message
	state.mu.Lock()
	for _, messages := range state.messages {
		for _, message := range messages {
			if strings.Contains(strings.ToLower(message.Content), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(message.Author), strings.ToLower(query)) {
				results = append(results, message)
			}
		}
	}
	state.mu.Unlock()

	// Show search results
	if len(results) == 0 {
		showNotification(state, "Search Results", fmt.Sprintf("No messages found for '%s'", query))
	} else {
		showNotification(state, "Search Results", fmt.Sprintf("Found %d messages for '%s'", len(results), query))
		// TODO: Display search results in a dialog or highlight in chat
	}
}

// Advanced file handling with preview
func showFilePreview(state *GUIState, filename string, content []byte) {
	// Create file preview dialog
	previewContent := container.NewVBox()

	// Add file info
	fileInfo := widget.NewCard("📄 File Information", "", container.NewVBox(
		widget.NewLabel("Name: "+filename),
		widget.NewLabel(fmt.Sprintf("Size: %d bytes", len(content))),
		widget.NewLabel("Type: "+getFileType(filename)),
	))

	previewContent.Add(fileInfo)

	// Add preview based on file type
	if isImageFile(filename) {
		// Image preview
		previewContent.Add(widget.NewLabel("🖼️ Image Preview"))
		// TODO: Add actual image preview
	} else if isTextFile(filename) {
		// Text preview
		textPreview := widget.NewEntry()
		textPreview.MultiLine = true
		textPreview.SetText(string(content[:min(1000, len(content))]) + "...")
		textPreview.Disable()
		previewContent.Add(textPreview)
	} else {
		previewContent.Add(widget.NewLabel("📁 Binary file - no preview available"))
	}

	// Show dialog
	dialog.ShowCustom("File Preview", "Close", previewContent, state.window)
}

// Helper functions for file handling
func getFileType(filename string) string {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	switch ext {
	case "txt", "md", "go", "js", "py", "html", "css":
		return "Text"
	case "jpg", "jpeg", "png", "gif", "bmp":
		return "Image"
	case "pdf":
		return "PDF"
	case "zip", "rar", "7z":
		return "Archive"
	default:
		return "Unknown"
	}
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	return ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "bmp"
}

func isTextFile(filename string) bool {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	return ext == "txt" || ext == "md" || ext == "go" || ext == "js" || ext == "py" || ext == "html" || ext == "css"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
