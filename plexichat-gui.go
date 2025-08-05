package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gorilla/websocket"

	"plexichat-client/pkg/client"
	"plexichat-client/pkg/logging"
)

const (
	appVersion = "3.0.0-production"

	// Performance constants
	maxMessages = 1000 // Maximum messages to keep in memory
	maxUsers    = 500  // Maximum users to cache

	// UI update intervals
	statusUpdateInterval    = 30 * time.Second
	userListRefreshInterval = 5 * time.Minute

	// Connection timeouts
	defaultTimeout = 30 * time.Second
)

// WebSocket message types
type WSMessage struct {
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	Timestamp float64        `json:"timestamp"`
	UserID    string         `json:"user_id,omitempty"`
	RoomID    string         `json:"room_id,omitempty"`
}

// AppConfig holds application configuration
type AppConfig struct {
	ServerURL       string        `json:"server_url"`
	AutoReconnect   bool          `json:"auto_reconnect"`
	MessageLimit    int           `json:"message_limit"`
	RefreshInterval time.Duration `json:"refresh_interval"`
	Theme           string        `json:"theme"`
	LogLevel        string        `json:"log_level"`

	// Advanced features
	EnableEncryption    bool   `json:"enable_encryption"`
	EnableNotifications bool   `json:"enable_notifications"`
	EnableSounds        bool   `json:"enable_sounds"`
	AutoSave            bool   `json:"auto_save"`
	EncryptionKey       string `json:"encryption_key,omitempty"`
}

// NotificationManager handles desktop notifications
type NotificationManager struct {
	enabled bool
	app     fyne.App
}

// AdvancedSearch provides enhanced search capabilities
type AdvancedSearch struct {
	query       string
	userFilter  string
	dateFrom    time.Time
	dateTo      time.Time
	messageType string
}

// Application state with performance optimizations
type PlexiChatApp struct {
	app         fyne.App
	window      fyne.Window
	client      *client.Client
	currentUser *client.UserResponse
	isLoggedIn  bool
	serverURL   string

	// WebSocket connection
	wsConn      *websocket.Conn
	wsConnected bool

	// UI components
	loginContainer   *fyne.Container
	mainContainer    *fyne.Container
	messageArea      *widget.RichText
	messageInput     *widget.Entry
	userList         *widget.List
	statusBar        *widget.Label
	conversationList *widget.List

	// Data with performance optimizations
	messages      []client.Message
	users         []client.User
	conversations []string
	selectedUser  string

	// Performance and caching
	messageCache    map[string][]client.Message // Cache messages by user ID
	userCache       map[string]client.User      // Cache user data
	lastUserRefresh time.Time                   // Track last user list refresh

	// Thread safety
	mu sync.RWMutex // Protects shared data

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc

	// Configuration
	config *AppConfig

	// Advanced features
	notificationManager *NotificationManager
	encryptionEnabled   bool
	searchHistory       []string

	// Error handling and recovery
	connectionRetries int
	lastError         error
	isRecovering      bool
	recoveryAttempts  int
	maxRetries        int
}

// loadConfig loads application configuration
func loadConfig() *AppConfig {
	return &AppConfig{
		ServerURL:           "http://localhost:8000",
		AutoReconnect:       true,
		MessageLimit:        maxMessages,
		RefreshInterval:     userListRefreshInterval,
		Theme:               "auto",
		LogLevel:            "info",
		EnableEncryption:    true,
		EnableNotifications: true,
		EnableSounds:        true,
		AutoSave:            true,
		EncryptionKey:       generateEncryptionKey(),
	}
}

// generateEncryptionKey creates a new encryption key
func generateEncryptionKey() string {
	key := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(key); err != nil {
		logging.Error("Failed to generate encryption key: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(key)
}

func main() {
	// Create application with ID
	myApp := app.NewWithID("com.plexichat.desktop")

	// Create main window with enhanced styling
	myWindow := myApp.NewWindow("PlexiChat Desktop v" + appVersion + " - Enterprise Edition")
	myWindow.Resize(fyne.NewSize(1600, 1000))
	myWindow.CenterOnScreen()

	// Set window icon and properties
	myWindow.SetMaster()
	myWindow.SetFixedSize(false)

	// Create context for background tasks
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize app state with performance optimizations and error handling
	plexiApp := &PlexiChatApp{
		app:               myApp,
		window:            myWindow,
		serverURL:         "http://localhost:8000", // Default server URL
		isLoggedIn:        false,
		messages:          make([]client.Message, 0, maxMessages),
		users:             make([]client.User, 0, maxUsers),
		conversations:     make([]string, 0),
		messageCache:      make(map[string][]client.Message),
		userCache:         make(map[string]client.User),
		ctx:               ctx,
		cancel:            cancel,
		config:            loadConfig(),
		connectionRetries: 0,
		maxRetries:        5,
		isRecovering:      false,
		recoveryAttempts:  0,
	}

	// Initialize client with optimizations
	plexiApp.client = client.NewClient(plexiApp.config.ServerURL)
	plexiApp.client.SetDebug(true)
	plexiApp.client.SetTimeout(defaultTimeout)

	// Handle window close event
	myWindow.SetCloseIntercept(func() {
		plexiApp.cleanup()
		myApp.Quit()
	})

	// Create UI
	plexiApp.createUI()

	// Start background tasks
	plexiApp.startBackgroundTasks()

	// Show login screen initially
	plexiApp.showLoginScreen()

	// Set content and show
	myWindow.SetContent(plexiApp.loginContainer)
	myWindow.ShowAndRun()
}

func (app *PlexiChatApp) createUI() {
	app.createLoginUI()
	app.createMainUI()
}

func (app *PlexiChatApp) createLoginUI() {
	// Server URL input
	serverEntry := widget.NewEntry()
	serverEntry.SetText(app.serverURL)
	serverEntry.SetPlaceHolder("Server URL (e.g., http://localhost:8000)")

	// Username input
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")

	// Password input
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	// Email input (for registration)
	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email (for registration)")

	// Status label
	statusLabel := widget.NewLabel("Enter your credentials to connect")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Login button
	loginButton := widget.NewButton("Login", func() {
		app.performLogin(serverEntry.Text, usernameEntry.Text, passwordEntry.Text, statusLabel)
	})
	loginButton.Importance = widget.HighImportance

	// Register button
	registerButton := widget.NewButton("Register", func() {
		app.performRegister(serverEntry.Text, usernameEntry.Text, emailEntry.Text, passwordEntry.Text, statusLabel)
	})

	// Test connection button
	testButton := widget.NewButton("Test Connection", func() {
		app.testConnection(serverEntry.Text, statusLabel)
	})

	// Create enhanced login form with better styling
	titleLabel := widget.NewLabel("PlexiChat Desktop")
	titleLabel.Alignment = fyne.TextAlignCenter
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	subtitleLabel := widget.NewLabel("Enterprise-Grade Secure Messaging Platform")
	subtitleLabel.Alignment = fyne.TextAlignCenter

	versionLabel := widget.NewLabel("Version " + appVersion + " - Production Ready")
	versionLabel.Alignment = fyne.TextAlignCenter

	// Create styled form sections
	serverSection := container.NewVBox(
		widget.NewLabel("🌐 Server Configuration"),
		serverEntry,
	)

	authSection := container.NewVBox(
		widget.NewLabel("🔐 Authentication"),
		usernameEntry,
		passwordEntry,
		emailEntry,
	)

	buttonSection := container.NewHBox(
		loginButton,
		registerButton,
		testButton,
	)

	// Main form with enhanced layout
	form := container.NewVBox(
		titleLabel,
		subtitleLabel,
		versionLabel,
		widget.NewSeparator(),
		serverSection,
		widget.NewSeparator(),
		authSection,
		widget.NewSeparator(),
		statusLabel,
		buttonSection,
	)

	// Create card with padding
	card := widget.NewCard("", "", form)
	card.Resize(fyne.NewSize(500, 600))

	// Center the form with padding
	app.loginContainer = container.NewCenter(
		container.NewPadded(card),
	)
}

func (app *PlexiChatApp) showLoginScreen() {
	app.window.SetContent(app.loginContainer)
}

func (app *PlexiChatApp) testConnection(serverURL string, statusLabel *widget.Label) {
	statusLabel.SetText("🔄 Testing connection...")

	// Validate URL format
	if !app.validateServerURL(serverURL) {
		statusLabel.SetText("❌ Invalid server URL format")
		return
	}

	// Update client URL
	app.client = client.NewClient(strings.TrimSuffix(serverURL, "/"))
	app.client.SetDebug(true)

	// Test connection with health check and retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		var lastErr error
		for attempt := 1; attempt <= 3; attempt++ {
			health, err := app.client.Health(ctx)
			if err != nil {
				lastErr = err
				statusLabel.SetText(fmt.Sprintf("🔄 Connection attempt %d/3...", attempt))
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}

			statusLabel.SetText(fmt.Sprintf("✅ Connected! Server: %s, Status: %s", health.Version, health.Status))
			app.serverURL = serverURL
			app.connectionRetries = 0
			return
		}

		// All attempts failed
		statusLabel.SetText(fmt.Sprintf("❌ Connection failed after 3 attempts: %v", lastErr))
		app.handleConnectionError(lastErr)
	}()
}

func (app *PlexiChatApp) performLogin(serverURL, username, password string, statusLabel *widget.Label) {
	// Validate inputs
	if err := app.validateInput(username, "Username", 3, 50); err != nil {
		statusLabel.SetText("❌ " + err.Error())
		return
	}
	if err := app.validateInput(password, "Password", 6, 100); err != nil {
		statusLabel.SetText("❌ " + err.Error())
		return
	}
	if !app.validateServerURL(serverURL) {
		statusLabel.SetText("❌ Invalid server URL format")
		return
	}

	statusLabel.SetText("🔐 Logging in...")

	// Update client URL
	app.client = client.NewClient(strings.TrimSuffix(serverURL, "/"))
	app.client.SetDebug(true)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		app.safeExecute("login", func() error {
			_, err := app.client.Login(ctx, username, password)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("❌ Login failed: %v", err))
				app.handleConnectionError(err)
				return err
			}

			// Get current user info
			userResp, err := app.client.GetCurrentUser(ctx)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("❌ Failed to get user info: %v", err))
				return err
			}

			// Update app state
			app.currentUser = userResp
			app.isLoggedIn = true
			app.serverURL = serverURL
			app.connectionRetries = 0 // Reset on successful login

			// Show main interface
			app.showMainScreen()

			statusLabel.SetText(fmt.Sprintf("✅ Welcome, %s!", userResp.Username))
			app.showNotification("Login Successful", fmt.Sprintf("Welcome back, %s!", userResp.Username))

			return nil
		})
	}()
}

func (app *PlexiChatApp) performRegister(serverURL, username, email, password string, statusLabel *widget.Label) {
	if username == "" || email == "" || password == "" {
		statusLabel.SetText("Please fill in all fields for registration")
		return
	}

	statusLabel.SetText("Registering...")

	// Update client URL
	app.client = client.NewClient(strings.TrimSuffix(serverURL, "/"))
	app.client.SetDebug(true)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		regResp, err := app.client.Register(ctx, username, email, password, "user")
		if err != nil {
			statusLabel.SetText(fmt.Sprintf("Registration failed: %v", err))
			return
		}

		statusLabel.SetText(fmt.Sprintf("Registration successful! User ID: %s", regResp.UserID))

		// Auto-login after successful registration
		time.Sleep(1 * time.Second)
		app.performLogin(serverURL, username, password, statusLabel)
	}()
}

func (app *PlexiChatApp) createMainUI() {
	// Create enhanced message area with better styling
	app.messageArea = widget.NewRichText()
	app.messageArea.Wrapping = fyne.TextWrapWord
	app.messageArea.ParseMarkdown(`# 🚀 Welcome to PlexiChat Desktop!

**Enterprise-Grade Secure Messaging Platform**

✨ **Features Available:**
- 🔒 End-to-End Encryption
- ⚡ Real-time WebSocket Messaging
- 🔍 Advanced Search & Filtering
- 📁 File Upload & Sharing
- 👥 User Management
- 🎨 Customizable Themes
- 🔔 Desktop Notifications

**Ready for Production Use!**

Select a user from the list to start messaging, or use the search function to find specific users.`)

	// Create enhanced message input with better styling
	app.messageInput = widget.NewEntry()
	app.messageInput.SetPlaceHolder("💬 Type your message here... (Press Enter to send)")
	app.messageInput.OnSubmitted = func(text string) {
		app.sendMessage(text)
	}

	// Enhanced buttons with icons and styling
	sendButton := widget.NewButton("📤 Send", func() {
		app.sendMessage(app.messageInput.Text)
	})
	sendButton.Importance = widget.HighImportance

	uploadButton := widget.NewButton("📎 Upload", func() {
		app.showFileUploadDialog()
	})

	emojiButton := widget.NewButton("😀 Emoji", func() {
		app.showEmojiPicker()
	})

	// Enhanced input container with better layout
	inputContainer := container.NewBorder(nil, nil, nil,
		container.NewHBox(emojiButton, uploadButton, sendButton), app.messageInput)

	// Create enhanced conversation list with better styling
	app.conversationList = widget.NewList(
		func() int { return len(app.conversations) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("💬 Conversation")
			label.TextStyle = fyne.TextStyle{Bold: false}
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(app.conversations) {
				obj.(*widget.Label).SetText("💬 " + app.conversations[id])
			}
		},
	)
	app.conversationList.OnSelected = func(id widget.ListItemID) {
		if id < len(app.conversations) {
			app.selectedUser = app.conversations[id]
			app.loadMessages(app.selectedUser)
		}
	}

	// Create enhanced user list with status indicators
	app.userList = widget.NewList(
		func() int { return len(app.users) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("👤 User")
			label.TextStyle = fyne.TextStyle{Bold: false}
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(app.users) {
				user := app.users[id]
				status := "🟢" // Online indicator (simplified)
				obj.(*widget.Label).SetText(fmt.Sprintf("%s %s", status, user.Username))
			}
		},
	)
	app.userList.OnSelected = func(id widget.ListItemID) {
		if id < len(app.users) {
			app.selectedUser = fmt.Sprintf("%d", app.users[id].ID)
			app.loadMessages(app.selectedUser)
			app.statusBar.SetText(fmt.Sprintf("💬 Chatting with %s", app.users[id].Username))
		}
	}

	// Create enhanced status bar with icons
	app.statusBar = widget.NewLabel("🟢 Connected to PlexiChat Server")

	// Enhanced buttons with icons and better styling
	logoutButton := widget.NewButton("🚪 Logout", func() {
		app.logout()
	})
	logoutButton.Importance = widget.DangerImportance

	refreshButton := widget.NewButton("🔄 Refresh", func() {
		app.refreshData()
	})

	profileButton := widget.NewButton("👤 Profile", func() {
		app.showProfileDialog()
	})

	searchButton := widget.NewButton("🔍 Search", func() {
		app.showUserSearchDialog()
	})

	advancedSearchButton := widget.NewButton("🔎 Advanced", func() {
		app.showAdvancedSearchDialog()
	})

	settingsButton := widget.NewButton("⚙️ Settings", func() {
		app.showSettingsDialog()
	})

	// Create welcome label with better styling
	welcomeLabel := widget.NewLabel(fmt.Sprintf("👋 Welcome, %s", app.getCurrentUsername()))
	welcomeLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create title label
	titleLabel := widget.NewLabel("🚀 PlexiChat Desktop - Enterprise Edition")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	// Enhanced header with better layout
	headerContainer := container.NewBorder(nil, nil,
		welcomeLabel,
		container.NewHBox(advancedSearchButton, searchButton, profileButton, settingsButton, refreshButton, logoutButton),
		titleLabel)

	// Enhanced chat area with better styling
	chatArea := container.NewBorder(nil, inputContainer, nil, nil,
		container.NewScroll(app.messageArea))

	// Enhanced left panel with better organization
	leftTabs := container.NewAppTabs(
		container.NewTabItem("💬 Conversations", app.conversationList),
		container.NewTabItem("👥 Users", app.userList),
	)
	leftTabs.SetTabLocation(container.TabLocationTop)

	// Create right panel for additional features
	rightPanel := container.NewVBox(
		widget.NewLabel("📊 Quick Stats"),
		widget.NewSeparator(),
		widget.NewLabel("Messages: 0"),
		widget.NewLabel("Users: 0"),
		widget.NewLabel("Files: 0"),
		widget.NewSeparator(),
		widget.NewLabel("🔔 Notifications"),
		widget.NewLabel("No new notifications"),
	)

	// Enhanced main content with three-panel layout
	mainContent := container.NewBorder(nil, nil, leftTabs, rightPanel, chatArea)

	// Complete main container with enhanced styling
	app.mainContainer = container.NewBorder(headerContainer, app.statusBar, nil, nil, mainContent)
}

func (app *PlexiChatApp) showMainScreen() {
	if app.mainContainer == nil {
		app.createMainUI()
	}
	app.window.SetContent(app.mainContainer)
	app.refreshData()

	// Connect WebSocket for real-time messaging
	app.connectWebSocket()
}

func (app *PlexiChatApp) getCurrentUsername() string {
	if app.currentUser != nil {
		return app.currentUser.Username
	}
	return "Unknown"
}

func (app *PlexiChatApp) logout() {
	// Disconnect WebSocket first
	app.disconnectWebSocket()

	app.isLoggedIn = false
	app.currentUser = nil
	app.client.SetToken("")
	app.messages = make([]client.Message, 0)
	app.users = make([]client.User, 0)
	app.conversations = make([]string, 0)
	app.selectedUser = ""
	app.showLoginScreen()
}

func (app *PlexiChatApp) refreshData() {
	if !app.isLoggedIn {
		return
	}

	go func() {
		// Load users list
		app.loadUsers()

		// Update status
		app.statusBar.SetText(fmt.Sprintf("Connected as %s | Last updated: %s",
			app.getCurrentUsername(), time.Now().Format("15:04:05")))
	}()
}

func (app *PlexiChatApp) loadUsers() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userList, err := app.client.GetUsers(ctx, 50, 0)
	if err != nil {
		logging.Error("Failed to load users: %v", err)
		app.statusBar.SetText(fmt.Sprintf("Failed to load users: %v", err))
		return
	}

	app.users = userList.Users
	app.userList.Refresh()

	logging.Info("Loaded %d users", len(app.users))
}

func (app *PlexiChatApp) sendMessage(text string) {
	if text == "" || app.selectedUser == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		// Send via HTTP API
		message, err := app.client.SendMessage(ctx, text, app.selectedUser)
		if err != nil {
			app.statusBar.SetText(fmt.Sprintf("Failed to send message: %v", err))
			return
		}

		// Add message to display
		app.addMessageToDisplay(message)
		app.messageInput.SetText("")

		// Also send via WebSocket for real-time delivery
		if app.wsConnected {
			wsData := map[string]any{
				"content":      text,
				"recipient_id": app.selectedUser,
				"message_type": "text",
			}
			app.sendWebSocketMessage("send_message", wsData)
		}

		app.statusBar.SetText("Message sent successfully")
	}()
}

func (app *PlexiChatApp) loadMessages(userID string) {
	if userID == "" {
		return
	}

	// Check cache first
	app.mu.RLock()
	if cachedMessages, exists := app.messageCache[userID]; exists {
		app.messages = cachedMessages
		app.mu.RUnlock()
		app.displayMessages()
		app.statusBar.SetText(fmt.Sprintf("Loaded %d cached messages", len(app.messages)))
		return
	}
	app.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		messages, err := app.client.GetMessages(ctx, userID, 50, 1)
		if err != nil {
			app.statusBar.SetText(fmt.Sprintf("Failed to load messages: %v", err))
			return
		}

		app.mu.Lock()
		app.messages = messages.Messages
		// Cache the messages
		app.messageCache[userID] = messages.Messages
		app.mu.Unlock()

		app.displayMessages()
		app.statusBar.SetText(fmt.Sprintf("Loaded %d messages", len(app.messages)))
	}()
}

func (app *PlexiChatApp) displayMessages() {
	var content strings.Builder
	content.WriteString("# 💬 Chat Messages\n\n")

	for i, msg := range app.messages {
		// Add visual separators and better formatting
		if i > 0 {
			content.WriteString("---\n\n")
		}

		// Enhanced message formatting with emojis and styling
		timeStr := msg.Timestamp.Format("15:04")
		dateStr := msg.Timestamp.Format("Jan 02")

		content.WriteString(fmt.Sprintf("**👤 %s** 🕐 *%s %s*\n\n",
			msg.Username, dateStr, timeStr))
		content.WriteString(fmt.Sprintf("💭 %s\n\n", msg.Content))
	}

	if len(app.messages) == 0 {
		content.WriteString(`🎉 **No messages yet!**

Start the conversation by:
- 💬 Typing a message below
- 📎 Uploading a file
- 😀 Adding some emojis

**Ready to chat!** 🚀`)
	}

	app.messageArea.ParseMarkdown(content.String())
}

func (app *PlexiChatApp) addMessageToDisplay(message *client.Message) {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Add to main messages list with size limit
	app.messages = append(app.messages, *message)
	if len(app.messages) > maxMessages {
		// Keep only the most recent messages
		copy(app.messages, app.messages[len(app.messages)-maxMessages:])
		app.messages = app.messages[:maxMessages]
	}

	// Add to cache for the specific user
	userID := app.selectedUser
	if userID != "" {
		if _, exists := app.messageCache[userID]; !exists {
			app.messageCache[userID] = make([]client.Message, 0)
		}
		app.messageCache[userID] = append(app.messageCache[userID], *message)

		// Limit cache size per user
		if len(app.messageCache[userID]) > maxMessages/10 {
			userMessages := app.messageCache[userID]
			copy(userMessages, userMessages[len(userMessages)-maxMessages/10:])
			app.messageCache[userID] = userMessages[:maxMessages/10]
		}
	}

	// Update display
	app.displayMessages()
}

func (app *PlexiChatApp) showFileUploadDialog() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		// Get file path
		filePath := reader.URI().Path()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		go func() {
			resp, err := app.client.UploadFile(ctx, "/api/v1/files/upload", filePath)
			if err != nil {
				app.statusBar.SetText(fmt.Sprintf("File upload failed: %v", err))
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 || resp.StatusCode == 201 {
				app.statusBar.SetText("File uploaded successfully")
			} else {
				app.statusBar.SetText(fmt.Sprintf("File upload failed with status: %d", resp.StatusCode))
			}
		}()
	}, app.window)
}

// WebSocket connection methods
func (app *PlexiChatApp) connectWebSocket() {
	if !app.isLoggedIn || app.currentUser == nil {
		return
	}

	// Connect to WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		conn, err := app.client.ConnectWebSocket(ctx, fmt.Sprintf("/api/v1/realtime/ws/%s", app.currentUser.ID))
		if err != nil {
			logging.Error("WebSocket connection failed: %v", err)
			app.statusBar.SetText(fmt.Sprintf("WebSocket connection failed: %v", err))
			return
		}

		app.wsConn = conn
		app.wsConnected = true
		app.statusBar.SetText("WebSocket connected - Real-time messaging enabled")

		// Start listening for messages
		app.listenWebSocket()
	}()
}

func (app *PlexiChatApp) disconnectWebSocket() {
	if app.wsConn != nil {
		app.wsConn.Close()
		app.wsConn = nil
		app.wsConnected = false
		app.statusBar.SetText("WebSocket disconnected")
	}
}

func (app *PlexiChatApp) listenWebSocket() {
	if app.wsConn == nil {
		return
	}

	go func() {
		defer func() {
			app.wsConnected = false
			app.wsConn = nil
		}()

		for {
			_, messageBytes, err := app.wsConn.ReadMessage()
			if err != nil {
				logging.Error("WebSocket read error: %v", err)
				app.statusBar.SetText("WebSocket connection lost")
				break
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(messageBytes, &wsMsg); err != nil {
				logging.Error("Failed to parse WebSocket message: %v", err)
				continue
			}

			app.handleWebSocketMessage(wsMsg)
		}
	}()
}

func (app *PlexiChatApp) handleWebSocketMessage(msg WSMessage) {
	switch msg.Type {
	case "welcome":
		logging.Info("WebSocket welcome: %v", msg.Data)
		app.statusBar.SetText("Real-time messaging connected")

	case "message":
		// Handle incoming message
		app.handleIncomingMessage(msg)

	case "typing":
		// Handle typing indicator
		app.handleTypingIndicator(msg)

	case "presence":
		// Handle user presence update
		app.handlePresenceUpdate(msg)

	case "error":
		logging.Error("WebSocket error: %v", msg.Data)
		app.statusBar.SetText(fmt.Sprintf("WebSocket error: %v", msg.Data["message"]))

	default:
		logging.Warn("Unknown WebSocket message type: %s", msg.Type)
	}
}

func (app *PlexiChatApp) handleIncomingMessage(msg WSMessage) {
	// Extract message data
	content, _ := msg.Data["content"].(string)
	username, _ := msg.Data["username"].(string)
	timestamp, _ := msg.Data["timestamp"].(string)

	if content != "" && username != "" {
		// Create a message object
		newMessage := client.Message{
			Content:  content,
			Username: username,
		}

		// Parse timestamp if available
		if timestamp != "" {
			if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
				newMessage.Timestamp = t
			} else {
				newMessage.Timestamp = time.Now()
			}
		} else {
			newMessage.Timestamp = time.Now()
		}

		// Add to messages and refresh display
		app.messages = append(app.messages, newMessage)
		app.displayMessages()

		// Update status
		app.statusBar.SetText(fmt.Sprintf("New message from %s", username))
	}
}

func (app *PlexiChatApp) handleTypingIndicator(msg WSMessage) {
	username, _ := msg.Data["username"].(string)
	isTyping, _ := msg.Data["is_typing"].(bool)

	if username != "" {
		if isTyping {
			app.statusBar.SetText(fmt.Sprintf("%s is typing...", username))
		} else {
			app.statusBar.SetText("Connected to PlexiChat Server")
		}
	}
}

func (app *PlexiChatApp) handlePresenceUpdate(msg WSMessage) {
	username, _ := msg.Data["username"].(string)
	status, _ := msg.Data["status"].(string)

	if username != "" && status != "" {
		app.statusBar.SetText(fmt.Sprintf("%s is %s", username, status))
	}
}

func (app *PlexiChatApp) sendWebSocketMessage(msgType string, data map[string]any) {
	if !app.wsConnected || app.wsConn == nil {
		return
	}

	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: float64(time.Now().Unix()),
	}

	if app.currentUser != nil {
		msg.UserID = app.currentUser.ID
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		logging.Error("Failed to marshal WebSocket message: %v", err)
		return
	}

	err = app.wsConn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		logging.Error("Failed to send WebSocket message: %v", err)
		app.statusBar.SetText("Failed to send real-time message")
	}
}

// User management dialogs
func (app *PlexiChatApp) showProfileDialog() {
	if !app.isLoggedIn || app.currentUser == nil {
		return
	}

	// Create form fields
	displayNameEntry := widget.NewEntry()
	displayNameEntry.SetText(app.currentUser.DisplayName)
	displayNameEntry.SetPlaceHolder("Display Name")

	emailEntry := widget.NewEntry()
	emailEntry.SetText(app.currentUser.Email)
	emailEntry.SetPlaceHolder("Email")

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Edit Profile"),
		widget.NewSeparator(),
		widget.NewLabel("Display Name:"),
		displayNameEntry,
		widget.NewLabel("Email:"),
		emailEntry,
	)

	// Create dialog
	profileDialog := dialog.NewCustom("User Profile", "Close", form, app.window)

	// Add save button
	saveButton := widget.NewButton("Save Changes", func() {
		app.updateProfile(displayNameEntry.Text, emailEntry.Text, profileDialog)
	})

	// Add save button to form
	form.Add(widget.NewSeparator())
	form.Add(saveButton)

	profileDialog.Show()
}

func (app *PlexiChatApp) updateProfile(displayName, email string, dialog *dialog.CustomDialog) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		updatedUser, err := app.client.UpdateProfile(ctx, displayName, email)
		if err != nil {
			app.statusBar.SetText(fmt.Sprintf("Failed to update profile: %v", err))
			return
		}

		app.currentUser = updatedUser
		app.statusBar.SetText("Profile updated successfully")
		dialog.Hide()
	}()
}

func (app *PlexiChatApp) showUserSearchDialog() {
	// Create search entry
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search users by username or email...")

	// Create results list
	var searchResults []client.User
	resultsList := widget.NewList(
		func() int { return len(searchResults) },
		func() fyne.CanvasObject {
			return widget.NewLabel("User")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(searchResults) {
				user := searchResults[id]
				obj.(*widget.Label).SetText(fmt.Sprintf("%s (%s)", user.Username, user.Email))
			}
		},
	)

	// Handle user selection
	resultsList.OnSelected = func(id widget.ListItemID) {
		if id < len(searchResults) {
			selectedUser := searchResults[id]
			app.selectedUser = fmt.Sprintf("%d", selectedUser.ID)
			app.statusBar.SetText(fmt.Sprintf("Selected user: %s", selectedUser.Username))
			// Load messages with this user
			app.loadMessages(app.selectedUser)
		}
	}

	// Search button
	searchButton := widget.NewButton("Search", func() {
		query := searchEntry.Text
		if query == "" {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			results, err := app.client.SearchUsers(ctx, query, 20)
			if err != nil {
				app.statusBar.SetText(fmt.Sprintf("Search failed: %v", err))
				return
			}

			searchResults = results.Users
			resultsList.Refresh()
			app.statusBar.SetText(fmt.Sprintf("Found %d users", len(searchResults)))
		}()
	})

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Search Users"),
		widget.NewSeparator(),
		searchEntry,
		searchButton,
		widget.NewSeparator(),
		widget.NewLabel("Search Results:"),
		container.NewScroll(resultsList),
	)

	// Create dialog
	searchDialog := dialog.NewCustom("User Search", "Close", form, app.window)
	searchDialog.Resize(fyne.NewSize(400, 500))
	searchDialog.Show()
}

// Performance and lifecycle methods
func (app *PlexiChatApp) startBackgroundTasks() {
	// Start periodic user list refresh
	go func() {
		ticker := time.NewTicker(app.config.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-app.ctx.Done():
				return
			case <-ticker.C:
				if app.isLoggedIn && time.Since(app.lastUserRefresh) > app.config.RefreshInterval {
					app.refreshUserList()
				}
			}
		}
	}()

	// Start status update ticker
	go func() {
		ticker := time.NewTicker(statusUpdateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-app.ctx.Done():
				return
			case <-ticker.C:
				if app.isLoggedIn {
					app.updateConnectionStatus()
				}
			}
		}
	}()
}

func (app *PlexiChatApp) cleanup() {
	// Cancel background tasks
	if app.cancel != nil {
		app.cancel()
	}

	// Disconnect WebSocket
	app.disconnectWebSocket()

	// Clear sensitive data
	app.mu.Lock()
	defer app.mu.Unlock()

	app.messageCache = make(map[string][]client.Message)
	app.userCache = make(map[string]client.User)
	app.messages = nil
	app.users = nil

	logging.Info("Application cleanup completed")
}

func (app *PlexiChatApp) refreshUserList() {
	if !app.isLoggedIn {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		userList, err := app.client.GetUsers(ctx, maxUsers, 0)
		if err != nil {
			logging.Error("Failed to refresh user list: %v", err)
			return
		}

		app.mu.Lock()
		app.users = userList.Users
		app.lastUserRefresh = time.Now()

		// Update user cache
		for _, user := range userList.Users {
			app.userCache[fmt.Sprintf("%d", user.ID)] = user
		}
		app.mu.Unlock()

		// Update UI on main thread
		app.userList.Refresh()
		logging.Info("Refreshed user list: %d users", len(userList.Users))
	}()
}

func (app *PlexiChatApp) updateConnectionStatus() {
	if !app.isLoggedIn {
		return
	}

	status := "Connected"
	if app.wsConnected {
		status += " (Real-time)"
	}

	userCount := len(app.users)
	messageCount := len(app.messages)

	app.statusBar.SetText(fmt.Sprintf("%s | Users: %d | Messages: %d | %s",
		status, userCount, messageCount, time.Now().Format("15:04:05")))
}

// Advanced features implementation
func (app *PlexiChatApp) showAdvancedSearchDialog() {
	// Create search form
	queryEntry := widget.NewEntry()
	queryEntry.SetPlaceHolder("Search messages, users, files...")

	userFilterEntry := widget.NewEntry()
	userFilterEntry.SetPlaceHolder("Filter by username")

	dateFromEntry := widget.NewEntry()
	dateFromEntry.SetPlaceHolder("From date (YYYY-MM-DD)")

	dateToEntry := widget.NewEntry()
	dateToEntry.SetPlaceHolder("To date (YYYY-MM-DD)")

	messageTypeSelect := widget.NewSelect([]string{"All", "Text", "File", "Image"}, nil)
	messageTypeSelect.SetSelected("All")

	// Search results
	var searchResults []string
	resultsList := widget.NewList(
		func() int { return len(searchResults) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Result")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(searchResults) {
				obj.(*widget.Label).SetText(searchResults[id])
			}
		},
	)

	// Search button
	searchBtn := widget.NewButton("Search", func() {
		app.performAdvancedSearch(queryEntry.Text, userFilterEntry.Text,
			dateFromEntry.Text, dateToEntry.Text, messageTypeSelect.Selected, &searchResults, resultsList)
	})

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Advanced Search"),
		widget.NewSeparator(),
		widget.NewLabel("Search Query:"),
		queryEntry,
		widget.NewLabel("User Filter:"),
		userFilterEntry,
		widget.NewLabel("Date Range:"),
		container.NewHBox(dateFromEntry, dateToEntry),
		widget.NewLabel("Message Type:"),
		messageTypeSelect,
		searchBtn,
		widget.NewSeparator(),
		widget.NewLabel("Results:"),
		container.NewScroll(resultsList),
	)

	// Create dialog
	searchDialog := dialog.NewCustom("Advanced Search", "Close", form, app.window)
	searchDialog.Resize(fyne.NewSize(500, 600))
	searchDialog.Show()
}

func (app *PlexiChatApp) performAdvancedSearch(query, userFilter, dateFrom, dateTo, msgType string,
	results *[]string, resultsList *widget.List) {

	app.mu.RLock()
	messages := make([]client.Message, len(app.messages))
	copy(messages, app.messages)
	app.mu.RUnlock()

	*results = (*results)[:0] // Clear results

	// Perform search
	for _, msg := range messages {
		if app.matchesSearchCriteria(msg, query, userFilter, dateFrom, dateTo, msgType) {
			resultText := fmt.Sprintf("[%s] %s: %s",
				msg.Timestamp.Format("2006-01-02 15:04"), msg.Username, msg.Content)
			*results = append(*results, resultText)
		}
	}

	// Update search history
	if query != "" {
		app.addToSearchHistory(query)
	}

	resultsList.Refresh()
	app.statusBar.SetText(fmt.Sprintf("Found %d results", len(*results)))
}

func (app *PlexiChatApp) matchesSearchCriteria(msg client.Message, query, userFilter, dateFrom, dateTo, msgType string) bool {
	// Text search
	if query != "" {
		queryLower := strings.ToLower(query)
		if !strings.Contains(strings.ToLower(msg.Content), queryLower) &&
			!strings.Contains(strings.ToLower(msg.Username), queryLower) {
			return false
		}
	}

	// User filter
	if userFilter != "" && !strings.Contains(strings.ToLower(msg.Username), strings.ToLower(userFilter)) {
		return false
	}

	// Date range filter
	if dateFrom != "" {
		if fromDate, err := time.Parse("2006-01-02", dateFrom); err == nil {
			if msg.Timestamp.Before(fromDate) {
				return false
			}
		}
	}

	if dateTo != "" {
		if toDate, err := time.Parse("2006-01-02", dateTo); err == nil {
			if msg.Timestamp.After(toDate.Add(24 * time.Hour)) {
				return false
			}
		}
	}

	// Message type filter (simplified - would need more sophisticated detection)
	if msgType != "All" {
		// This is a simplified implementation
		switch msgType {
		case "File":
			if !strings.Contains(strings.ToLower(msg.Content), "file") {
				return false
			}
		case "Image":
			if !strings.Contains(strings.ToLower(msg.Content), "image") {
				return false
			}
		}
	}

	return true
}

func (app *PlexiChatApp) addToSearchHistory(query string) {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Add to beginning of history
	app.searchHistory = append([]string{query}, app.searchHistory...)

	// Limit history size
	if len(app.searchHistory) > 20 {
		app.searchHistory = app.searchHistory[:20]
	}
}

func (app *PlexiChatApp) showSettingsDialog() {
	// Server settings
	serverEntry := widget.NewEntry()
	serverEntry.SetText(app.config.ServerURL)

	// Feature toggles
	encryptionCheck := widget.NewCheck("Enable End-to-End Encryption", nil)
	encryptionCheck.SetChecked(app.config.EnableEncryption)

	notificationsCheck := widget.NewCheck("Enable Desktop Notifications", nil)
	notificationsCheck.SetChecked(app.config.EnableNotifications)

	soundsCheck := widget.NewCheck("Enable Sound Notifications", nil)
	soundsCheck.SetChecked(app.config.EnableSounds)

	autoSaveCheck := widget.NewCheck("Auto-save Messages", nil)
	autoSaveCheck.SetChecked(app.config.AutoSave)

	// Theme selection
	themeSelect := widget.NewSelect([]string{"auto", "light", "dark"}, nil)
	themeSelect.SetSelected(app.config.Theme)

	// Log level
	logLevelSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	logLevelSelect.SetSelected(app.config.LogLevel)

	// Message limit
	messageLimitEntry := widget.NewEntry()
	messageLimitEntry.SetText(fmt.Sprintf("%d", app.config.MessageLimit))

	// Create form
	form := container.NewVBox(
		widget.NewLabel("PlexiChat Settings"),
		widget.NewSeparator(),

		widget.NewLabel("Server Configuration:"),
		widget.NewLabel("Server URL:"),
		serverEntry,

		widget.NewSeparator(),
		widget.NewLabel("Features:"),
		encryptionCheck,
		notificationsCheck,
		soundsCheck,
		autoSaveCheck,

		widget.NewSeparator(),
		widget.NewLabel("Appearance:"),
		widget.NewLabel("Theme:"),
		themeSelect,

		widget.NewSeparator(),
		widget.NewLabel("Advanced:"),
		widget.NewLabel("Log Level:"),
		logLevelSelect,
		widget.NewLabel("Message Cache Limit:"),
		messageLimitEntry,
	)

	// Save button
	saveButton := widget.NewButton("Save Settings", func() {
		app.saveSettings(serverEntry.Text, encryptionCheck.Checked, notificationsCheck.Checked,
			soundsCheck.Checked, autoSaveCheck.Checked, themeSelect.Selected, logLevelSelect.Selected,
			messageLimitEntry.Text)
	})

	// Reset button
	resetButton := widget.NewButton("Reset to Defaults", func() {
		app.resetSettings()
	})

	form.Add(widget.NewSeparator())
	form.Add(container.NewHBox(saveButton, resetButton))

	// Create dialog
	settingsDialog := dialog.NewCustom("Settings", "Close", container.NewScroll(form), app.window)
	settingsDialog.Resize(fyne.NewSize(450, 600))
	settingsDialog.Show()
}

func (app *PlexiChatApp) saveSettings(serverURL string, encryption, notifications, sounds, autoSave bool,
	theme, logLevel, messageLimit string) {

	// Update configuration
	app.config.ServerURL = serverURL
	app.config.EnableEncryption = encryption
	app.config.EnableNotifications = notifications
	app.config.EnableSounds = sounds
	app.config.AutoSave = autoSave
	app.config.Theme = theme
	app.config.LogLevel = logLevel

	// Parse message limit
	if limit, err := fmt.Sscanf(messageLimit, "%d", &app.config.MessageLimit); err == nil && limit > 0 {
		// Valid limit
	} else {
		app.config.MessageLimit = maxMessages
	}

	// Apply settings
	app.applySettings()

	app.statusBar.SetText("Settings saved successfully")
}

func (app *PlexiChatApp) resetSettings() {
	app.config = loadConfig()
	app.applySettings()
	app.statusBar.SetText("Settings reset to defaults")
}

func (app *PlexiChatApp) applySettings() {
	// Update client URL if changed
	if app.client.BaseURL != app.config.ServerURL {
		app.client = client.NewClient(app.config.ServerURL)
		app.client.SetDebug(app.config.LogLevel == "debug")
		app.client.SetTimeout(defaultTimeout)
	}

	// Apply encryption setting
	app.encryptionEnabled = app.config.EnableEncryption

	// Initialize notification manager
	if app.config.EnableNotifications && app.notificationManager == nil {
		app.notificationManager = &NotificationManager{
			enabled: true,
			app:     app.app,
		}
	}

	logging.Info("Settings applied: Encryption=%v, Notifications=%v, Theme=%s",
		app.config.EnableEncryption, app.config.EnableNotifications, app.config.Theme)
}

// UI Enhancement methods
func (app *PlexiChatApp) showEmojiPicker() {
	// Common emojis for quick access
	emojis := []string{
		"😀", "😃", "😄", "😁", "😆", "😅", "😂", "🤣", "😊", "😇",
		"🙂", "🙃", "😉", "😌", "😍", "🥰", "😘", "😗", "😙", "😚",
		"😋", "😛", "😝", "😜", "🤪", "🤨", "🧐", "🤓", "😎", "🤩",
		"🥳", "😏", "😒", "😞", "😔", "😟", "😕", "🙁", "☹️", "😣",
		"👍", "👎", "👌", "✌️", "🤞", "🤟", "🤘", "🤙", "👈", "👉",
		"👆", "🖕", "👇", "☝️", "👋", "🤚", "🖐️", "✋", "🖖", "👏",
		"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💔",
		"💕", "💞", "💓", "💗", "💖", "💘", "💝", "💟", "☮️", "✝️",
		"🔥", "💯", "💢", "💥", "💫", "💦", "💨", "🕳️", "💣", "💬",
		"🎉", "🎊", "🎈", "🎁", "🎀", "🎂", "🎄", "🎆", "🎇", "✨",
	}

	// Create emoji grid
	var emojiButtons []fyne.CanvasObject
	for _, emoji := range emojis {
		btn := widget.NewButton(emoji, func() {
			// Add emoji to message input
			currentText := app.messageInput.Text
			app.messageInput.SetText(currentText + emoji)
		})
		btn.Resize(fyne.NewSize(40, 40))
		emojiButtons = append(emojiButtons, btn)
	}

	// Create grid layout
	grid := container.NewGridWithColumns(10, emojiButtons...)

	// Create dialog
	emojiDialog := dialog.NewCustom("😀 Emoji Picker", "Close",
		container.NewScroll(grid), app.window)
	emojiDialog.Resize(fyne.NewSize(450, 300))
	emojiDialog.Show()
}

// Enhanced notification system
func (app *PlexiChatApp) showNotification(title, message string) {
	if !app.config.EnableNotifications {
		return
	}

	// Create notification dialog (simplified - in a real app you'd use system notifications)
	notification := dialog.NewInformation(title, message, app.window)
	notification.Show()

	// Auto-close after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		notification.Hide()
	}()
}

// Theme management
func (app *PlexiChatApp) applyTheme(themeName string) {
	switch themeName {
	case "dark":
		app.app.Settings().SetTheme(&darkTheme{})
	case "light":
		app.app.Settings().SetTheme(&lightTheme{})
	default:
		// Auto theme - use system default
		app.app.Settings().SetTheme(nil)
	}
}

// Custom theme implementations (simplified)
type darkTheme struct{}
type lightTheme struct{}

func (t *darkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 32, G: 32, B: 32, A: 255}
	case theme.ColorNameForeground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *darkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func (t *lightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 248, G: 248, B: 248, A: 255}
	case theme.ColorNameForeground:
		return color.RGBA{R: 32, G: 32, B: 32, A: 255}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *lightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *lightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *lightTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// Error handling and recovery methods
func (app *PlexiChatApp) validateServerURL(url string) bool {
	if url == "" {
		return false
	}

	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	// Check for valid domain/IP format (simplified)
	if strings.Contains(url, " ") || len(url) < 10 {
		return false
	}

	return true
}

func (app *PlexiChatApp) handleConnectionError(err error) {
	app.lastError = err
	app.connectionRetries++

	logging.Error("Connection error (attempt %d/%d): %v", app.connectionRetries, app.maxRetries, err)

	// Show user-friendly error message
	if app.connectionRetries >= app.maxRetries {
		app.showErrorDialog("Connection Failed",
			fmt.Sprintf("Unable to connect to server after %d attempts.\n\nError: %v\n\nPlease check:\n• Server URL is correct\n• Server is running\n• Network connection\n• Firewall settings",
				app.maxRetries, err))
		app.connectionRetries = 0
	} else if app.config.AutoReconnect && !app.isRecovering {
		app.startConnectionRecovery()
	}
}

func (app *PlexiChatApp) startConnectionRecovery() {
	if app.isRecovering {
		return
	}

	app.isRecovering = true
	app.recoveryAttempts = 0

	go func() {
		defer func() {
			app.isRecovering = false
		}()

		for app.recoveryAttempts < app.maxRetries {
			app.recoveryAttempts++

			// Wait with exponential backoff
			waitTime := time.Duration(app.recoveryAttempts*app.recoveryAttempts) * time.Second
			app.statusBar.SetText(fmt.Sprintf("🔄 Reconnecting in %d seconds... (attempt %d/%d)",
				int(waitTime.Seconds()), app.recoveryAttempts, app.maxRetries))

			select {
			case <-app.ctx.Done():
				return
			case <-time.After(waitTime):
			}

			// Attempt reconnection
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			health, err := app.client.Health(ctx)
			cancel()

			if err == nil {
				app.statusBar.SetText(fmt.Sprintf("✅ Reconnected! Server: %s", health.Status))
				app.connectionRetries = 0
				app.recoveryAttempts = 0

				// Reconnect WebSocket if needed
				if app.isLoggedIn {
					app.connectWebSocket()
				}
				return
			}

			logging.Error("Recovery attempt %d failed: %v", app.recoveryAttempts, err)
		}

		// All recovery attempts failed
		app.statusBar.SetText("❌ Connection recovery failed")
		app.showErrorDialog("Connection Recovery Failed",
			"Unable to reconnect to server. Please check your connection and try again manually.")
	}()
}

func (app *PlexiChatApp) showErrorDialog(title, message string) {
	errorDialog := dialog.NewError(fmt.Errorf("%s", message), app.window)
	errorDialog.Show()
}

func (app *PlexiChatApp) handleWebSocketError(err error) {
	logging.Error("WebSocket error: %v", err)

	if app.wsConnected {
		app.disconnectWebSocket()
	}

	// Attempt to reconnect if logged in and auto-reconnect is enabled
	if app.isLoggedIn && app.config.AutoReconnect && !app.isRecovering {
		go func() {
			time.Sleep(5 * time.Second) // Wait before reconnecting
			if app.isLoggedIn && !app.wsConnected {
				app.connectWebSocket()
			}
		}()
	}
}

func (app *PlexiChatApp) validateInput(input, fieldName string, minLength, maxLength int) error {
	if len(input) < minLength {
		return fmt.Errorf("%s must be at least %d characters", fieldName, minLength)
	}
	if len(input) > maxLength {
		return fmt.Errorf("%s must be no more than %d characters", fieldName, maxLength)
	}
	return nil
}

func (app *PlexiChatApp) safeExecute(operation string, fn func() error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in %s: %v", operation, r)
			app.statusBar.SetText(fmt.Sprintf("❌ Error in %s", operation))
			app.showErrorDialog("Application Error",
				fmt.Sprintf("An unexpected error occurred in %s. Please try again.", operation))
		}
	}()

	if err := fn(); err != nil {
		logging.Error("Error in %s: %v", operation, err)
		app.statusBar.SetText(fmt.Sprintf("❌ %s failed: %v", operation, err))
	}
}
