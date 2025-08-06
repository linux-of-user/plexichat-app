package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plexichat-client/pkg/logging"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ThemeType represents different theme types
type ThemeType string

const (
	ThemeLight        ThemeType = "light"
	ThemeDark         ThemeType = "dark"
	ThemeAuto         ThemeType = "auto"
	ThemeCustom       ThemeType = "custom"
	ThemeHighContrast ThemeType = "high_contrast"
	ThemeColorBlind   ThemeType = "colorblind"
)

// ColorScheme represents a complete color scheme
type ColorScheme struct {
	Name         string `json:"name"`
	Primary      string `json:"primary"`
	Secondary    string `json:"secondary"`
	Background   string `json:"background"`
	Surface      string `json:"surface"`
	Error        string `json:"error"`
	Warning      string `json:"warning"`
	Success      string `json:"success"`
	Info         string `json:"info"`
	OnPrimary    string `json:"on_primary"`
	OnSecondary  string `json:"on_secondary"`
	OnBackground string `json:"on_background"`
	OnSurface    string `json:"on_surface"`
	OnError      string `json:"on_error"`
	Accent       string `json:"accent"`
	Disabled     string `json:"disabled"`
	Hover        string `json:"hover"`
	Focus        string `json:"focus"`
	Selection    string `json:"selection"`
	Border       string `json:"border"`
	Shadow       string `json:"shadow"`
}

// Typography represents font and text styling
type Typography struct {
	FontFamily    string  `json:"font_family"`
	FontSize      float32 `json:"font_size"`
	LineHeight    float32 `json:"line_height"`
	LetterSpacing float32 `json:"letter_spacing"`
	FontWeight    string  `json:"font_weight"`
	HeadingScale  float32 `json:"heading_scale"`
	MonospaceFont string  `json:"monospace_font"`
	EmojiFont     string  `json:"emoji_font"`
}

// Animation represents animation settings
type Animation struct {
	Duration       time.Duration `json:"duration"`
	Easing         string        `json:"easing"`
	Enabled        bool          `json:"enabled"`
	ReducedMotion  bool          `json:"reduced_motion"`
	TransitionType string        `json:"transition_type"`
}

// Spacing represents spacing and sizing values
type Spacing struct {
	Tiny    float32 `json:"tiny"`
	Small   float32 `json:"small"`
	Medium  float32 `json:"medium"`
	Large   float32 `json:"large"`
	XLarge  float32 `json:"xlarge"`
	Padding struct {
		Tiny   float32 `json:"tiny"`
		Small  float32 `json:"small"`
		Medium float32 `json:"medium"`
		Large  float32 `json:"large"`
	} `json:"padding"`
	Margin struct {
		Tiny   float32 `json:"tiny"`
		Small  float32 `json:"small"`
		Medium float32 `json:"medium"`
		Large  float32 `json:"large"`
	} `json:"margin"`
	BorderRadius struct {
		Small  float32 `json:"small"`
		Medium float32 `json:"medium"`
		Large  float32 `json:"large"`
		Round  float32 `json:"round"`
	} `json:"border_radius"`
}

// ThemeConfig represents a complete theme configuration
type ThemeConfig struct {
	Name        string      `json:"name"`
	Type        ThemeType   `json:"type"`
	Version     string      `json:"version"`
	Author      string      `json:"author"`
	Description string      `json:"description"`
	Colors      ColorScheme `json:"colors"`
	Typography  Typography  `json:"typography"`
	Animation   Animation   `json:"animation"`
	Spacing     Spacing     `json:"spacing"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// PlexiChatTheme implements fyne.Theme with advanced customization
type PlexiChatTheme struct {
	config *ThemeConfig
	logger *logging.Logger
	mu     sync.RWMutex
}

// NewPlexiChatTheme creates a new theme instance
func NewPlexiChatTheme(config *ThemeConfig) *PlexiChatTheme {
	if config == nil {
		config = GetDefaultTheme()
	}

	return &PlexiChatTheme{
		config: config,
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// Color returns theme colors
func (t *PlexiChatTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	t.mu.RLock()
	defer t.mu.RUnlock()

	switch name {
	case theme.ColorNamePrimary:
		return parseColor(t.config.Colors.Primary)
	case theme.ColorNameBackground:
		return parseColor(t.config.Colors.Background)
	case theme.ColorNameButton:
		return parseColor(t.config.Colors.Secondary)
	case theme.ColorNameDisabled:
		return parseColor(t.config.Colors.Disabled)
	case theme.ColorNameError:
		return parseColor(t.config.Colors.Error)
	case theme.ColorNameFocus:
		return parseColor(t.config.Colors.Focus)
	case theme.ColorNameForeground:
		return parseColor(t.config.Colors.OnBackground)
	case theme.ColorNameHover:
		return parseColor(t.config.Colors.Hover)
	case theme.ColorNameInputBackground:
		return parseColor(t.config.Colors.Surface)
	case theme.ColorNamePlaceHolder:
		return parseColor(t.config.Colors.Disabled)
	case theme.ColorNamePressed:
		return parseColor(t.config.Colors.Accent)
	case theme.ColorNameScrollBar:
		return parseColor(t.config.Colors.Border)
	case theme.ColorNameSelection:
		return parseColor(t.config.Colors.Selection)
	case theme.ColorNameSeparator:
		return parseColor(t.config.Colors.Border)
	case theme.ColorNameShadow:
		return parseColor(t.config.Colors.Shadow)
	case theme.ColorNameSuccess:
		return parseColor(t.config.Colors.Success)
	case theme.ColorNameWarning:
		return parseColor(t.config.Colors.Warning)
	default:
		// Fallback to default theme
		return theme.DefaultTheme().Color(name, variant)
	}
}

// Font returns theme fonts
func (t *PlexiChatTheme) Font(style fyne.TextStyle) fyne.Resource {
	// For now, return default fonts
	// In a full implementation, this would load custom fonts
	return theme.DefaultTheme().Font(style)
}

// Icon returns theme icons
func (t *PlexiChatTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	// For now, return default icons
	// In a full implementation, this would load custom icon sets
	return theme.DefaultTheme().Icon(name)
}

// Size returns theme sizes
func (t *PlexiChatTheme) Size(name fyne.ThemeSizeName) float32 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	switch name {
	case theme.SizeNameText:
		return t.config.Typography.FontSize
	case theme.SizeNamePadding:
		return t.config.Spacing.Padding.Medium
	case theme.SizeNameScrollBar:
		return t.config.Spacing.Small
	case theme.SizeNameScrollBarSmall:
		return t.config.Spacing.Tiny
	case theme.SizeNameSeparatorThickness:
		return 1.0
	case theme.SizeNameInputBorder:
		return 2.0
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// parseColor converts hex color string to color.Color
func parseColor(hexColor string) color.Color {
	if len(hexColor) != 7 || hexColor[0] != '#' {
		return color.Black // Fallback
	}

	var r, g, b uint8
	fmt.Sscanf(hexColor[1:], "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// GetDefaultTheme returns the default theme configuration
func GetDefaultTheme() *ThemeConfig {
	return &ThemeConfig{
		Name:        "PlexiChat Default",
		Type:        ThemeLight,
		Version:     "1.0.0",
		Author:      "PlexiChat Team",
		Description: "Default PlexiChat theme with modern design",
		Colors: ColorScheme{
			Name:         "Default Light",
			Primary:      "#2196F3",
			Secondary:    "#03DAC6",
			Background:   "#FFFFFF",
			Surface:      "#F5F5F5",
			Error:        "#F44336",
			Warning:      "#FF9800",
			Success:      "#4CAF50",
			Info:         "#2196F3",
			OnPrimary:    "#FFFFFF",
			OnSecondary:  "#000000",
			OnBackground: "#000000",
			OnSurface:    "#000000",
			OnError:      "#FFFFFF",
			Accent:       "#FF4081",
			Disabled:     "#BDBDBD",
			Hover:        "#E3F2FD",
			Focus:        "#1976D2",
			Selection:    "#E3F2FD",
			Border:       "#E0E0E0",
			Shadow:       "#00000020",
		},
		Typography: Typography{
			FontFamily:    "Roboto",
			FontSize:      14,
			LineHeight:    1.4,
			LetterSpacing: 0,
			FontWeight:    "normal",
			HeadingScale:  1.2,
			MonospaceFont: "Roboto Mono",
			EmojiFont:     "Noto Color Emoji",
		},
		Animation: Animation{
			Duration:       200 * time.Millisecond,
			Easing:         "ease-in-out",
			Enabled:        true,
			ReducedMotion:  false,
			TransitionType: "fade",
		},
		Spacing: Spacing{
			Tiny:   4,
			Small:  8,
			Medium: 16,
			Large:  24,
			XLarge: 32,
			Padding: struct {
				Tiny   float32 `json:"tiny"`
				Small  float32 `json:"small"`
				Medium float32 `json:"medium"`
				Large  float32 `json:"large"`
			}{
				Tiny:   4,
				Small:  8,
				Medium: 16,
				Large:  24,
			},
			Margin: struct {
				Tiny   float32 `json:"tiny"`
				Small  float32 `json:"small"`
				Medium float32 `json:"medium"`
				Large  float32 `json:"large"`
			}{
				Tiny:   4,
				Small:  8,
				Medium: 16,
				Large:  24,
			},
			BorderRadius: struct {
				Small  float32 `json:"small"`
				Medium float32 `json:"medium"`
				Large  float32 `json:"large"`
				Round  float32 `json:"round"`
			}{
				Small:  4,
				Medium: 8,
				Large:  16,
				Round:  50,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// GetDarkTheme returns a dark theme configuration
func GetDarkTheme() *ThemeConfig {
	config := GetDefaultTheme()
	config.Name = "PlexiChat Dark"
	config.Type = ThemeDark
	config.Description = "Dark theme for PlexiChat with modern design"

	// Update colors for dark theme
	config.Colors = ColorScheme{
		Name:         "Default Dark",
		Primary:      "#BB86FC",
		Secondary:    "#03DAC6",
		Background:   "#121212",
		Surface:      "#1E1E1E",
		Error:        "#CF6679",
		Warning:      "#FFB74D",
		Success:      "#81C784",
		Info:         "#64B5F6",
		OnPrimary:    "#000000",
		OnSecondary:  "#000000",
		OnBackground: "#FFFFFF",
		OnSurface:    "#FFFFFF",
		OnError:      "#000000",
		Accent:       "#FF4081",
		Disabled:     "#616161",
		Hover:        "#2C2C2C",
		Focus:        "#BB86FC",
		Selection:    "#3F3F3F",
		Border:       "#3F3F3F",
		Shadow:       "#00000040",
	}

	return config
}

// ThemeManager manages theme loading, saving, and switching
type ThemeManager struct {
	currentTheme *PlexiChatTheme
	themes       map[string]*ThemeConfig
	configDir    string
	logger       *logging.Logger
	mu           sync.RWMutex
}

// NewThemeManager creates a new theme manager
func NewThemeManager(configDir string) *ThemeManager {
	tm := &ThemeManager{
		themes:    make(map[string]*ThemeConfig),
		configDir: configDir,
		logger:    logging.NewLogger(logging.INFO, nil, true),
	}

	// Load default themes
	tm.themes["default"] = GetDefaultTheme()
	tm.themes["dark"] = GetDarkTheme()

	// Set default theme
	tm.currentTheme = NewPlexiChatTheme(tm.themes["default"])

	// Load custom themes
	tm.loadCustomThemes()

	return tm
}

// GetCurrentTheme returns the current theme
func (tm *ThemeManager) GetCurrentTheme() *PlexiChatTheme {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.currentTheme
}

// SetTheme switches to a different theme
func (tm *ThemeManager) SetTheme(themeName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	config, exists := tm.themes[themeName]
	if !exists {
		return fmt.Errorf("theme %s not found", themeName)
	}

	tm.currentTheme = NewPlexiChatTheme(config)
	tm.logger.Info("Switched to theme: %s", themeName)

	return nil
}

// GetAvailableThemes returns list of available themes
func (tm *ThemeManager) GetAvailableThemes() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var themes []string
	for name := range tm.themes {
		themes = append(themes, name)
	}

	return themes
}

// SaveTheme saves a custom theme
func (tm *ThemeManager) SaveTheme(config *ThemeConfig) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	config.UpdatedAt = time.Now()
	tm.themes[config.Name] = config

	// Save to file
	themePath := filepath.Join(tm.configDir, "themes", config.Name+".json")
	if err := os.MkdirAll(filepath.Dir(themePath), 0755); err != nil {
		return fmt.Errorf("failed to create themes directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	if err := os.WriteFile(themePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save theme: %w", err)
	}

	tm.logger.Info("Saved theme: %s", config.Name)
	return nil
}

// loadCustomThemes loads custom themes from disk
func (tm *ThemeManager) loadCustomThemes() {
	themesDir := filepath.Join(tm.configDir, "themes")

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		// Directory doesn't exist yet, that's okay
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			themePath := filepath.Join(themesDir, entry.Name())

			data, err := os.ReadFile(themePath)
			if err != nil {
				tm.logger.Error("Failed to read theme file %s: %v", themePath, err)
				continue
			}

			var config ThemeConfig
			if err := json.Unmarshal(data, &config); err != nil {
				tm.logger.Error("Failed to parse theme file %s: %v", themePath, err)
				continue
			}

			tm.themes[config.Name] = &config
			tm.logger.Debug("Loaded custom theme: %s", config.Name)
		}
	}
}
