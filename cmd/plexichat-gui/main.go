package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

plexichat-client/pkg/analytics
	"plexichat-client/pkg/api"
	"plexichat-client/pkg/auth"
	"plexichat-client/pkg/cache"
	"plexichat-client/pkg/collaboration"
	"plexichat-client/pkg/config"
	"plexichat-client/pkg/files"
	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/notifications"
	"plexichat-client/pkg/plugins"
	"plexichat-client/pkg/security"
	"plexichat-client/pkg/ui"
)

// Application represents the main PlexiChat GUI application
type Application struct {
	config               *config.Config
	logger               *logging.Logger
	apiClient            *api.Client
	authManager          *auth.Manager
	cacheManager         *cache.Manager
	securityManager      *security.Manager
	uiManager            *ui.Manager
	fileManager          *files.FileManager
	pluginManager        *plugins.PluginManager
	analyticsManager     *analytics.Analytics
	notificationManager  *notifications.NotificationManager
	collaborationManager *collaboration.CollaborationManager

	ctx                  context.Context
	cancel               context.CancelFunc
}

// NewApplication creates a new PlexiChat GUI application instance
func NewApplication(configPath string) (*Application, error) {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	logger := logging.NewLogger(logging.INFO, nil, true)
	logger.Info("Starting PlexiChat GUI Client v%s", cfg.App.Version)

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize components
	if err := app.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return app, nil
}

// initializeComponents initializes all application components
func (app *Application) initializeComponents() error {
	app.logger.Info("Initializing application components...")

	// Initialize cache manager
	cacheConfig := &cache.CacheConfig{
		Type:            "memory",
		MaxSize:         100 * 1024 * 1024, // 100MB
		TTL:             30 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
	app.cacheManager = cache.NewManager(cacheConfig)

	// Initialize security manager
	securityConfig := &security.SecurityConfig{
		EncryptionEnabled: true,
		HashAlgorithm:     "bcrypt",
		TokenExpiry:       24 * time.Hour,
		MaxLoginAttempts:  5,
		LockoutDuration:   15 * time.Minute,
		TwoFactorEnabled:  false,
		SessionTimeout:    2 * time.Hour,
	}
	app.securityManager = security.NewManager(securityConfig)

	// Initialize API client
	apiConfig := &api.Config{
		BaseURL:       app.config.API.BaseURL,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		EnableLogging: true,
		EnableMetrics: true,
		EnableCaching: true,
		CacheManager:  app.cacheManager,
	}
	app.apiClient = api.NewClient(apiConfig)

	// Initialize auth manager
	authConfig := &auth.Config{
		TokenStorage:     "file",
		TokenPath:        filepath.Join(app.config.App.DataDir, "tokens"),
		RefreshThreshold: 5 * time.Minute,
		AutoRefresh:      true,
		SecureStorage:    true,
	}
	app.authManager = auth.NewManager(authConfig, app.apiClient, app.securityManager)

	// Initialize UI manager
	themeConfig := &ui.ThemeConfig{
		Name: "default",
		Colors: ui.ColorScheme{
			Primary:     "#007bff",
			Secondary:   "#6c757d",
			Success:     "#28a745",
			Warning:     "#ffc107",
			Error:       "#dc3545",
			Background:  "#ffffff",
			Surface:     "#f8f9fa",
			OnPrimary:   "#ffffff",
			OnSecondary: "#ffffff",
			OnSurface:   "#212529",
			Border:      "#dee2e6",
			Shadow:      "#00000020",
		},
		Typography: ui.Typography{
			FontFamily: "Inter, sans-serif",
			FontSize:   14,
			FontWeight: "400",
		},
		Spacing: ui.Spacing{
			Padding: ui.PaddingConfig{
				Small:  8,
				Medium: 16,
				Large:  24,
			},
			Margin: ui.MarginConfig{
				Small:  4,
				Medium: 8,
				Large:  16,
			},
			BorderRadius: ui.BorderRadiusConfig{
				Small:  4,
				Medium: 8,
				Large:  12,
			},
		},
	}
	app.uiManager = ui.NewManager(themeConfig)

	// Initialize file manager
	fileConfig := &files.FileManagerConfig{
		StorageDir:         filepath.Join(app.config.App.DataDir, "files"),
		ThumbnailDir:       filepath.Join(app.config.App.DataDir, "thumbnails"),
		PreviewDir:         filepath.Join(app.config.App.DataDir, "previews"),
		TempDir:            filepath.Join(app.config.App.DataDir, "temp"),
		MaxFileSize:        100 * 1024 * 1024, // 100MB
		AllowedTypes:       []string{"image/*", "text/*", "application/pdf"},
		GenerateThumbnails: true,
		GeneratePreviews:   true,
		VirusScanEnabled:   false,
		VersioningEnabled:  true,
		MaxVersions:        10,
		CleanupInterval:    24 * time.Hour,
		RetentionDays:      30,
		ChunkSize:          1024 * 1024, // 1MB
		ConcurrentUploads:  5,
	}
	app.fileManager = files.NewFileManager(fileConfig)

	// Initialize plugin manager
	pluginDir := filepath.Join(app.config.App.DataDir, "plugins")
	app.pluginManager = plugins.NewPluginManager(pluginDir)

	// Initialize analytics
	analyticsConfig := &analytics.AnalyticsConfig{
		Enabled:             true,
		SamplingRate:        1.0,
		BatchSize:           100,
		FlushInterval:       30 * time.Second,
		RetentionDays:       30,
		AnonymizeData:       true,
		CollectPerformance:  true,
		CollectErrors:       true,
		CollectFeatureUsage: true,
		ExportFormat:        "json",
		StorageDir:          filepath.Join(app.config.App.DataDir, "analytics"),
	}
	app.analyticsManager = analytics.NewAnalytics(analyticsConfig)

	// Initialize notifications
	notificationConfig := &notifications.NotificationConfig{
		Enabled:          true,
		DefaultChannel:   "desktop",
		Channels:         make(map[string]*notifications.NotificationChannel),
		GlobalFilters:    make([]*notifications.NotificationFilter, 0),
		DoNotDisturb:     false,
		BadgeCount:       true,
		Sounds:           true,
		Vibration:        true,
		StorageDir:       filepath.Join(app.config.App.DataDir, "notifications"),
		RetentionDays:    30,
		MaxNotifications: 1000,
		GroupSimilar:     true,
		GroupTimeWindow:  5 * time.Minute,
	}
	app.notificationManager = notifications.NewNotificationManager(notificationConfig)

	// Initialize collaboration manager
	app.collaborationManager = collaboration.NewCollaborationManager()



	app.logger.Info("All components initialized successfully")
	return nil
}

// Run starts the GUI application
func (app *Application) Run() error {
	app.logger.Info("Starting PlexiChat GUI application...")

	// Start analytics session
	sessionID := app.analyticsManager.StartSession("user_001")
	app.logger.Info("Started analytics session: %s", sessionID)

	// Track application start
	app.analyticsManager.TrackEvent(
		analytics.EventSystemEvent,
		"application",
		"start",
		map[string]interface{}{
			"version": app.config.App.Version,
			"mode":    "gui",
		},
	)

	// Send startup notification
	app.notificationManager.SendSimple(
		notifications.NotificationInfo,
		"PlexiChat Started",
		"PlexiChat GUI client has started successfully",
	)



	// Start UI
	app.logger.Info("Starting graphical user interface...")
	if err := app.startGUI(); err != nil {
		return fmt.Errorf("failed to start GUI: %w", err)
	}

	// Wait for shutdown signal
	app.waitForShutdown()

	return nil
}

// startGUI starts the graphical user interface
func (app *Application) startGUI() error {
	// For now, just log that GUI would start
	// In a real implementation, this would initialize the GUI framework (Fyne, etc.)
	app.logger.Info("GUI started (placeholder mode)")

	// Track UI start
	app.analyticsManager.TrackEvent(
		analytics.EventSystemEvent,
		"ui",
		"start",
		map[string]interface{}{
			"mode":      "gui",
			"framework": "placeholder",
		},
	)

	// Simulate GUI running
	app.logger.Info("GUI is running. Press Ctrl+C to exit.")

	return nil
}

// waitForShutdown waits for shutdown signals
func (app *Application) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		app.logger.Info("Received signal: %v", sig)
	case <-app.ctx.Done():
		app.logger.Info("Application context cancelled")
	}

	app.shutdown()
}

// shutdown gracefully shuts down the application
func (app *Application) shutdown() {
	app.logger.Info("Shutting down PlexiChat GUI application...")

	// Track application shutdown
	app.analyticsManager.TrackEvent(
		analytics.EventSystemEvent,
		"application",
		"shutdown",
		map[string]interface{}{
			"graceful": true,
			"mode":     "gui",
		},
	)

	// Send shutdown notification
	app.notificationManager.SendSimple(
		notifications.NotificationInfo,
		"PlexiChat Shutting Down",
		"PlexiChat GUI client is shutting down",
	)

	// Shutdown components in reverse order

	if app.collaborationManager != nil {
		app.collaborationManager.Shutdown()
	}

	if app.notificationManager != nil {
		app.notificationManager.Shutdown()
	}

	if app.analyticsManager != nil {
		app.analyticsManager.Shutdown()
	}

	if app.pluginManager != nil {
		// Unload all plugins
		for _, plugin := range app.pluginManager.ListPlugins() {
			app.pluginManager.UnloadPlugin(plugin.Manifest.Name)
		}
	}

	if app.fileManager != nil {
		app.fileManager.Shutdown()
	}

	if app.uiManager != nil {
		app.uiManager.Shutdown()
	}

	if app.authManager != nil {
		app.authManager.Shutdown()
	}

	if app.cacheManager != nil {
		app.cacheManager.Shutdown()
	}

	// Cancel application context
	app.cancel()

	app.logger.Info("PlexiChat GUI application shutdown complete")
}

// main is the entry point of the GUI application
func main() {
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		version    = flag.Bool("version", false, "Show version information")
		help       = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *help {
		fmt.Println("PlexiChat GUI Client - Modern Chat Application")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  plexichat-gui [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Println("PlexiChat GUI Client vb.1.1-97")
		fmt.Println("Build: 2024-01-01")
		fmt.Println("Go version:", "go1.21")
		return
	}

	// Create and run GUI application
	app, err := NewApplication(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create GUI application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "GUI application error: %v\n", err)
		os.Exit(1)
	}
}
