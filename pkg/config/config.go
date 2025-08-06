package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Config represents the application configuration
type Config struct {
	// Application settings
	App AppConfig `yaml:"app" json:"app"`

	// Server settings
	Server ServerConfig `yaml:"server" json:"server"`

	// Database settings
	Database DatabaseConfig `yaml:"database" json:"database"`

	// Security settings
	Security SecurityConfig `yaml:"security" json:"security"`

	// Logging settings
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// UI settings
	UI UIConfig `yaml:"ui" json:"ui"`

	// Features settings
	Features FeaturesConfig `yaml:"features" json:"features"`

	// Performance settings
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// Notification settings
	Notifications NotificationConfig `yaml:"notifications" json:"notifications"`

	// File upload settings
	FileUpload FileUploadConfig `yaml:"file_upload" json:"file_upload"`

	mu sync.RWMutex
}

// AppConfig contains application-specific settings
type AppConfig struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Environment string `yaml:"environment" json:"environment"`
	Debug       bool   `yaml:"debug" json:"debug"`
	DataDir     string `yaml:"data_dir" json:"data_dir"`
	ConfigDir   string `yaml:"config_dir" json:"config_dir"`
	LogDir      string `yaml:"log_dir" json:"log_dir"`
	TempDir     string `yaml:"temp_dir" json:"temp_dir"`
}

// ServerConfig contains server connection settings
type ServerConfig struct {
	URL            string        `yaml:"url" json:"url"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
	KeepAlive      bool          `yaml:"keep_alive" json:"keep_alive"`
	MaxConnections int           `yaml:"max_connections" json:"max_connections"`
	TLSEnabled     bool          `yaml:"tls_enabled" json:"tls_enabled"`
	TLSSkipVerify  bool          `yaml:"tls_skip_verify" json:"tls_skip_verify"`
	WebSocketURL   string        `yaml:"websocket_url" json:"websocket_url"`
	APIKey         string        `yaml:"api_key" json:"api_key"`
	UserAgent      string        `yaml:"user_agent" json:"user_agent"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path            string        `yaml:"path" json:"path"`
	MaxConnections  int           `yaml:"max_connections" json:"max_connections"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	BackupEnabled   bool          `yaml:"backup_enabled" json:"backup_enabled"`
	BackupInterval  time.Duration `yaml:"backup_interval" json:"backup_interval"`
	BackupRetention int           `yaml:"backup_retention" json:"backup_retention"`
	VacuumEnabled   bool          `yaml:"vacuum_enabled" json:"vacuum_enabled"`
	VacuumInterval  time.Duration `yaml:"vacuum_interval" json:"vacuum_interval"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	EncryptionEnabled bool          `yaml:"encryption_enabled" json:"encryption_enabled"`
	EncryptionKey     string        `yaml:"encryption_key" json:"encryption_key"`
	HashAlgorithm     string        `yaml:"hash_algorithm" json:"hash_algorithm"`
	TokenExpiry       time.Duration `yaml:"token_expiry" json:"token_expiry"`
	MaxLoginAttempts  int           `yaml:"max_login_attempts" json:"max_login_attempts"`
	LockoutDuration   time.Duration `yaml:"lockout_duration" json:"lockout_duration"`
	TwoFactorEnabled  bool          `yaml:"two_factor_enabled" json:"two_factor_enabled"`
	SessionTimeout    time.Duration `yaml:"session_timeout" json:"session_timeout"`
	RateLimitEnabled  bool          `yaml:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitRequests int           `yaml:"rate_limit_requests" json:"rate_limit_requests"`
	RateLimitWindow   time.Duration `yaml:"rate_limit_window" json:"rate_limit_window"`
	ContentValidation bool          `yaml:"content_validation" json:"content_validation"`
	IPWhitelist       []string      `yaml:"ip_whitelist" json:"ip_whitelist"`
	IPBlacklist       []string      `yaml:"ip_blacklist" json:"ip_blacklist"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level         string `yaml:"level" json:"level"`
	Format        string `yaml:"format" json:"format"`
	Output        string `yaml:"output" json:"output"`
	File          string `yaml:"file" json:"file"`
	MaxSize       int    `yaml:"max_size" json:"max_size"`
	MaxBackups    int    `yaml:"max_backups" json:"max_backups"`
	MaxAge        int    `yaml:"max_age" json:"max_age"`
	Compress      bool   `yaml:"compress" json:"compress"`
	EnableConsole bool   `yaml:"enable_console" json:"enable_console"`
	EnableFile    bool   `yaml:"enable_file" json:"enable_file"`
	EnableSyslog  bool   `yaml:"enable_syslog" json:"enable_syslog"`
	SyslogNetwork string `yaml:"syslog_network" json:"syslog_network"`
	SyslogAddress string `yaml:"syslog_address" json:"syslog_address"`
}

// UIConfig contains user interface settings
type UIConfig struct {
	Theme             string            `yaml:"theme" json:"theme"`
	Language          string            `yaml:"language" json:"language"`
	FontSize          int               `yaml:"font_size" json:"font_size"`
	FontFamily        string            `yaml:"font_family" json:"font_family"`
	ShowTimestamps    bool              `yaml:"show_timestamps" json:"show_timestamps"`
	ShowAvatars       bool              `yaml:"show_avatars" json:"show_avatars"`
	CompactMode       bool              `yaml:"compact_mode" json:"compact_mode"`
	AnimationsEnabled bool              `yaml:"animations_enabled" json:"animations_enabled"`
	SoundEnabled      bool              `yaml:"sound_enabled" json:"sound_enabled"`
	NotificationSound string            `yaml:"notification_sound" json:"notification_sound"`
	WindowWidth       int               `yaml:"window_width" json:"window_width"`
	WindowHeight      int               `yaml:"window_height" json:"window_height"`
	WindowMaximized   bool              `yaml:"window_maximized" json:"window_maximized"`
	CustomCSS         string            `yaml:"custom_css" json:"custom_css"`
	KeyboardShortcuts map[string]string `yaml:"keyboard_shortcuts" json:"keyboard_shortcuts"`
}

// FeaturesConfig contains feature toggle settings
type FeaturesConfig struct {
	FileUpload       bool `yaml:"file_upload" json:"file_upload"`
	FileDownload     bool `yaml:"file_download" json:"file_download"`
	ImagePreview     bool `yaml:"image_preview" json:"image_preview"`
	VideoPreview     bool `yaml:"video_preview" json:"video_preview"`
	AudioPreview     bool `yaml:"audio_preview" json:"audio_preview"`
	MessageEdit      bool `yaml:"message_edit" json:"message_edit"`
	MessageDelete    bool `yaml:"message_delete" json:"message_delete"`
	MessageReactions bool `yaml:"message_reactions" json:"message_reactions"`
	MessageThreads   bool `yaml:"message_threads" json:"message_threads"`
	TypingIndicators bool `yaml:"typing_indicators" json:"typing_indicators"`
	ReadReceipts     bool `yaml:"read_receipts" json:"read_receipts"`
	OnlineStatus     bool `yaml:"online_status" json:"online_status"`
	UserProfiles     bool `yaml:"user_profiles" json:"user_profiles"`
	ChannelHistory   bool `yaml:"channel_history" json:"channel_history"`
	SearchMessages   bool `yaml:"search_messages" json:"search_messages"`
	Notifications    bool `yaml:"notifications" json:"notifications"`
	Analytics        bool `yaml:"analytics" json:"analytics"`
	Plugins          bool `yaml:"plugins" json:"plugins"`
	Collaboration    bool `yaml:"collaboration" json:"collaboration"`
	AutoUpdate       bool `yaml:"auto_update" json:"auto_update"`
}

// PerformanceConfig contains performance settings
type PerformanceConfig struct {
	CacheEnabled       bool          `yaml:"cache_enabled" json:"cache_enabled"`
	CacheSize          int           `yaml:"cache_size" json:"cache_size"`
	CacheTTL           time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	MessageBatchSize   int           `yaml:"message_batch_size" json:"message_batch_size"`
	MessageBufferSize  int           `yaml:"message_buffer_size" json:"message_buffer_size"`
	ConnectionPoolSize int           `yaml:"connection_pool_size" json:"connection_pool_size"`
	WorkerPoolSize     int           `yaml:"worker_pool_size" json:"worker_pool_size"`
	GCInterval         time.Duration `yaml:"gc_interval" json:"gc_interval"`
	MemoryLimit        int64         `yaml:"memory_limit" json:"memory_limit"`
	CPULimit           float64       `yaml:"cpu_limit" json:"cpu_limit"`
	DiskCacheEnabled   bool          `yaml:"disk_cache_enabled" json:"disk_cache_enabled"`
	DiskCacheSize      int64         `yaml:"disk_cache_size" json:"disk_cache_size"`
}

// NotificationConfig contains notification settings
type NotificationConfig struct {
	Enabled       bool               `yaml:"enabled" json:"enabled"`
	Desktop       bool               `yaml:"desktop" json:"desktop"`
	Sound         bool               `yaml:"sound" json:"sound"`
	Email         bool               `yaml:"email" json:"email"`
	Push          bool               `yaml:"push" json:"push"`
	Channels      []string           `yaml:"channels" json:"channels"`
	Keywords      []string           `yaml:"keywords" json:"keywords"`
	MentionOnly   bool               `yaml:"mention_only" json:"mention_only"`
	QuietHours    bool               `yaml:"quiet_hours" json:"quiet_hours"`
	QuietStart    string             `yaml:"quiet_start" json:"quiet_start"`
	QuietEnd      string             `yaml:"quiet_end" json:"quiet_end"`
	EmailSettings EmailConfig        `yaml:"email_settings" json:"email_settings"`
	PushSettings  PushConfig         `yaml:"push_settings" json:"push_settings"`
	CustomRules   []NotificationRule `yaml:"custom_rules" json:"custom_rules"`
}

// EmailConfig contains email notification settings
type EmailConfig struct {
	SMTPHost     string `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort     int    `yaml:"smtp_port" json:"smtp_port"`
	SMTPUsername string `yaml:"smtp_username" json:"smtp_username"`
	SMTPPassword string `yaml:"smtp_password" json:"smtp_password"`
	FromAddress  string `yaml:"from_address" json:"from_address"`
	ToAddress    string `yaml:"to_address" json:"to_address"`
	TLSEnabled   bool   `yaml:"tls_enabled" json:"tls_enabled"`
}

// PushConfig contains push notification settings
type PushConfig struct {
	ServiceURL string            `yaml:"service_url" json:"service_url"`
	APIKey     string            `yaml:"api_key" json:"api_key"`
	DeviceID   string            `yaml:"device_id" json:"device_id"`
	Headers    map[string]string `yaml:"headers" json:"headers"`
}

// NotificationRule represents a custom notification rule
type NotificationRule struct {
	Name     string   `yaml:"name" json:"name"`
	Channels []string `yaml:"channels" json:"channels"`
	Keywords []string `yaml:"keywords" json:"keywords"`
	Users    []string `yaml:"users" json:"users"`
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Sound    string   `yaml:"sound" json:"sound"`
	Desktop  bool     `yaml:"desktop" json:"desktop"`
	Email    bool     `yaml:"email" json:"email"`
	Push     bool     `yaml:"push" json:"push"`
}

// FileUploadConfig contains file upload settings
type FileUploadConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	MaxFileSize     int64         `yaml:"max_file_size" json:"max_file_size"`
	MaxTotalSize    int64         `yaml:"max_total_size" json:"max_total_size"`
	AllowedTypes    []string      `yaml:"allowed_types" json:"allowed_types"`
	BlockedTypes    []string      `yaml:"blocked_types" json:"blocked_types"`
	ScanForViruses  bool          `yaml:"scan_for_viruses" json:"scan_for_viruses"`
	GenerateThumbs  bool          `yaml:"generate_thumbnails" json:"generate_thumbnails"`
	CompressImages  bool          `yaml:"compress_images" json:"compress_images"`
	StoragePath     string        `yaml:"storage_path" json:"storage_path"`
	TempPath        string        `yaml:"temp_path" json:"temp_path"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
	RetentionPeriod time.Duration `yaml:"retention_period" json:"retention_period"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:        "PlexiChat Client",
			Version:     "3.0.0-production",
			Environment: "production",
			Debug:       false,
			DataDir:     getDefaultDataDir(),
			ConfigDir:   getDefaultConfigDir(),
			LogDir:      getDefaultLogDir(),
			TempDir:     getDefaultTempDir(),
		},
		Server: ServerConfig{
			URL:            "http://localhost:8000",
			Timeout:        30 * time.Second,
			RetryAttempts:  3,
			RetryDelay:     5 * time.Second,
			KeepAlive:      true,
			MaxConnections: 10,
			TLSEnabled:     false,
			TLSSkipVerify:  false,
			WebSocketURL:   "ws://localhost:8000/ws",
			UserAgent:      "PlexiChat-Client/3.0.0",
		},
		Database: DatabaseConfig{
			Path:            filepath.Join(getDefaultDataDir(), "plexichat.db"),
			MaxConnections:  25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			BackupEnabled:   true,
			BackupInterval:  24 * time.Hour,
			BackupRetention: 7,
			VacuumEnabled:   true,
			VacuumInterval:  7 * 24 * time.Hour,
		},
		Security: SecurityConfig{
			EncryptionEnabled: true,
			HashAlgorithm:     "bcrypt",
			TokenExpiry:       24 * time.Hour,
			MaxLoginAttempts:  5,
			LockoutDuration:   15 * time.Minute,
			TwoFactorEnabled:  false,
			SessionTimeout:    2 * time.Hour,
			RateLimitEnabled:  true,
			RateLimitRequests: 60,
			RateLimitWindow:   time.Minute,
			ContentValidation: true,
		},
		Logging: LoggingConfig{
			Level:         "info",
			Format:        "json",
			Output:        "both",
			File:          filepath.Join(getDefaultLogDir(), "plexichat.log"),
			MaxSize:       100,
			MaxBackups:    5,
			MaxAge:        30,
			Compress:      true,
			EnableConsole: true,
			EnableFile:    true,
			EnableSyslog:  false,
		},
		UI: UIConfig{
			Theme:             "dark",
			Language:          "en",
			FontSize:          14,
			FontFamily:        "system-ui",
			ShowTimestamps:    true,
			ShowAvatars:       true,
			CompactMode:       false,
			AnimationsEnabled: true,
			SoundEnabled:      true,
			WindowWidth:       1200,
			WindowHeight:      800,
			WindowMaximized:   false,
			KeyboardShortcuts: getDefaultKeyboardShortcuts(),
		},
		Features: FeaturesConfig{
			FileUpload:       true,
			FileDownload:     true,
			ImagePreview:     true,
			VideoPreview:     true,
			AudioPreview:     true,
			MessageEdit:      true,
			MessageDelete:    true,
			MessageReactions: true,
			MessageThreads:   true,
			TypingIndicators: true,
			ReadReceipts:     true,
			OnlineStatus:     true,
			UserProfiles:     true,
			ChannelHistory:   true,
			SearchMessages:   true,
			Notifications:    true,
			Analytics:        true,
			Plugins:          true,
			Collaboration:    true,
			AutoUpdate:       true,
		},
		Performance: PerformanceConfig{
			CacheEnabled:       true,
			CacheSize:          1000,
			CacheTTL:           5 * time.Minute,
			MessageBatchSize:   50,
			MessageBufferSize:  1000,
			ConnectionPoolSize: 10,
			WorkerPoolSize:     5,
			GCInterval:         5 * time.Minute,
			MemoryLimit:        512 * 1024 * 1024, // 512MB
			CPULimit:           0.8,
			DiskCacheEnabled:   true,
			DiskCacheSize:      100 * 1024 * 1024, // 100MB
		},
		Notifications: NotificationConfig{
			Enabled:     true,
			Desktop:     true,
			Sound:       true,
			Email:       false,
			Push:        false,
			MentionOnly: false,
			QuietHours:  false,
			QuietStart:  "22:00",
			QuietEnd:    "08:00",
		},
		FileUpload: FileUploadConfig{
			Enabled:         true,
			MaxFileSize:     10 * 1024 * 1024,  // 10MB
			MaxTotalSize:    100 * 1024 * 1024, // 100MB
			AllowedTypes:    []string{"image/*", "text/*", "application/pdf"},
			ScanForViruses:  false,
			GenerateThumbs:  true,
			CompressImages:  true,
			StoragePath:     filepath.Join(getDefaultDataDir(), "files"),
			TempPath:        getDefaultTempDir(),
			CleanupInterval: 24 * time.Hour,
			RetentionPeriod: 30 * 24 * time.Hour,
		},
	}
}

// Helper functions for default paths
func getDefaultDataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".plexichat-app", "data")
}

func getDefaultConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".plexichat-app", "config")
}

func getDefaultLogDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".plexichat-app", "logs")
}

func getDefaultTempDir() string {
	return filepath.Join(os.TempDir(), "plexichat-app")
}

func getDefaultKeyboardShortcuts() map[string]string {
	return map[string]string{
		"send_message":    "Ctrl+Enter",
		"new_channel":     "Ctrl+N",
		"search":          "Ctrl+F",
		"upload_file":     "Ctrl+U",
		"toggle_sidebar":  "Ctrl+B",
		"next_channel":    "Ctrl+Tab",
		"prev_channel":    "Ctrl+Shift+Tab",
		"edit_message":    "Up",
		"delete_message":  "Delete",
		"reply_message":   "Ctrl+R",
		"mention_user":    "@",
		"emoji_picker":    "Ctrl+E",
		"toggle_mute":     "Ctrl+M",
		"zoom_in":         "Ctrl+=",
		"zoom_out":        "Ctrl+-",
		"reset_zoom":      "Ctrl+0",
		"quit":            "Ctrl+Q",
		"preferences":     "Ctrl+,",
		"help":            "F1",
		"developer_tools": "F12",
	}
}
