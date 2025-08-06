package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// PluginType represents different types of plugins
type PluginType string

const (
	PluginTypeIntegration    PluginType = "integration"
	PluginTypeTheme          PluginType = "theme"
	PluginTypeCommand        PluginType = "command"
	PluginTypeNotification   PluginType = "notification"
	PluginTypeAuthentication PluginType = "authentication"
	PluginTypeStorage        PluginType = "storage"
	PluginTypeTransport      PluginType = "transport"
	PluginTypeFilter         PluginType = "filter"
	PluginTypeBot            PluginType = "bot"
	PluginTypeAnalytics      PluginType = "analytics"
)

// PluginStatus represents plugin status
type PluginStatus string

const (
	StatusLoaded   PluginStatus = "loaded"
	StatusActive   PluginStatus = "active"
	StatusInactive PluginStatus = "inactive"
	StatusError    PluginStatus = "error"
	StatusUnloaded PluginStatus = "unloaded"
)

// PluginManifest describes a plugin
type PluginManifest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	License      string                 `json:"license"`
	Homepage     string                 `json:"homepage"`
	Type         PluginType             `json:"type"`
	EntryPoint   string                 `json:"entry_point"`
	Dependencies []string               `json:"dependencies"`
	Permissions  []string               `json:"permissions"`
	Config       map[string]interface{} `json:"config"`
	MinVersion   string                 `json:"min_version"`
	MaxVersion   string                 `json:"max_version"`
	Tags         []string               `json:"tags"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Plugin represents a loaded plugin
type Plugin struct {
	Manifest  *PluginManifest        `json:"manifest"`
	Status    PluginStatus           `json:"status"`
	Instance  PluginInterface        `json:"-"`
	LoadedAt  time.Time              `json:"loaded_at"`
	LastError string                 `json:"last_error,omitempty"`
	Config    map[string]interface{} `json:"config"`
	Metrics   *PluginMetrics         `json:"metrics"`
}

// PluginMetrics tracks plugin performance
type PluginMetrics struct {
	CallCount       int64         `json:"call_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	ErrorCount      int64         `json:"error_count"`
	LastCall        time.Time     `json:"last_call"`
	LastError       time.Time     `json:"last_error"`
}

// PluginInterface defines the interface all plugins must implement
type PluginInterface interface {
	// Initialize initializes the plugin with configuration
	Initialize(config map[string]interface{}) error

	// Start starts the plugin
	Start(ctx context.Context) error

	// Stop stops the plugin
	Stop() error

	// GetInfo returns plugin information
	GetInfo() *PluginInfo

	// HandleEvent handles events from the application
	HandleEvent(event *PluginEvent) error

	// GetCommands returns commands provided by this plugin
	GetCommands() []*PluginCommand

	// ExecuteCommand executes a plugin command
	ExecuteCommand(command string, args map[string]interface{}) (interface{}, error)

	// GetConfigSchema returns the configuration schema
	GetConfigSchema() map[string]interface{}

	// Validate validates the plugin configuration
	Validate(config map[string]interface{}) error
}

// PluginInfo contains basic plugin information
type PluginInfo struct {
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	Type        PluginType `json:"type"`
}

// PluginEvent represents events sent to plugins
type PluginEvent struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Context   context.Context        `json:"-"`
}

// PluginCommand represents a command provided by a plugin
type PluginCommand struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Usage       string             `json:"usage"`
	Parameters  []CommandParameter `json:"parameters"`
	Category    string             `json:"category"`
	Hidden      bool               `json:"hidden"`
}

// CommandParameter represents a command parameter
type CommandParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
	Choices     []string    `json:"choices,omitempty"`
}

// PluginManager manages plugin lifecycle
type PluginManager struct {
	plugins   map[string]*Plugin
	pluginDir string
	logger    *logging.Logger
	mu        sync.RWMutex
	eventBus  *EventBus
	registry  *PluginRegistry
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{
		plugins:   make(map[string]*Plugin),
		pluginDir: pluginDir,
		logger:    logging.NewLogger(logging.INFO, nil, true),
		eventBus:  NewEventBus(),
		registry:  NewPluginRegistry(),
	}
}

// LoadPlugin loads a plugin from a directory
func (pm *PluginManager) LoadPlugin(pluginPath string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Read manifest
	manifestPath := filepath.Join(pluginPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Check if plugin already loaded
	if _, exists := pm.plugins[manifest.Name]; exists {
		return fmt.Errorf("plugin %s already loaded", manifest.Name)
	}

	// Load plugin binary
	pluginBinary := filepath.Join(pluginPath, manifest.EntryPoint)
	p, err := plugin.Open(pluginBinary)
	if err != nil {
		return fmt.Errorf("failed to load plugin binary: %w", err)
	}

	// Get plugin instance
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin does not export NewPlugin function: %w", err)
	}

	newPluginFunc, ok := symbol.(func() PluginInterface)
	if !ok {
		return fmt.Errorf("NewPlugin function has wrong signature")
	}

	instance := newPluginFunc()

	// Create plugin
	pluginObj := &Plugin{
		Manifest: &manifest,
		Status:   StatusLoaded,
		Instance: instance,
		LoadedAt: time.Now(),
		Config:   manifest.Config,
		Metrics:  &PluginMetrics{},
	}

	// Initialize plugin
	if err := instance.Initialize(manifest.Config); err != nil {
		pluginObj.Status = StatusError
		pluginObj.LastError = err.Error()
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	pm.plugins[manifest.Name] = pluginObj
	pm.logger.Info("Loaded plugin: %s v%s", manifest.Name, manifest.Version)

	return nil
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin if running
	if plugin.Status == StatusActive {
		if err := plugin.Instance.Stop(); err != nil {
			pm.logger.Error("Error stopping plugin %s: %v", name, err)
		}
	}

	plugin.Status = StatusUnloaded
	delete(pm.plugins, name)

	pm.logger.Info("Unloaded plugin: %s", name)
	return nil
}

// StartPlugin starts a plugin
func (pm *PluginManager) StartPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if plugin.Status == StatusActive {
		return fmt.Errorf("plugin %s already active", name)
	}

	ctx := context.Background()
	if err := plugin.Instance.Start(ctx); err != nil {
		plugin.Status = StatusError
		plugin.LastError = err.Error()
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	plugin.Status = StatusActive
	pm.logger.Info("Started plugin: %s", name)

	return nil
}

// StopPlugin stops a plugin
func (pm *PluginManager) StopPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if plugin.Status != StatusActive {
		return fmt.Errorf("plugin %s not active", name)
	}

	if err := plugin.Instance.Stop(); err != nil {
		plugin.Status = StatusError
		plugin.LastError = err.Error()
		return fmt.Errorf("failed to stop plugin: %w", err)
	}

	plugin.Status = StatusInactive
	pm.logger.Info("Stopped plugin: %s", name)

	return nil
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (*Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns all loaded plugins
func (pm *PluginManager) ListPlugins() map[string]*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make(map[string]*Plugin)
	for name, plugin := range pm.plugins {
		plugins[name] = plugin
	}

	return plugins
}

// ExecuteCommand executes a plugin command
func (pm *PluginManager) ExecuteCommand(pluginName, command string, args map[string]interface{}) (interface{}, error) {
	pm.mu.RLock()
	plugin, exists := pm.plugins[pluginName]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	if plugin.Status != StatusActive {
		return nil, fmt.Errorf("plugin %s not active", pluginName)
	}

	// Update metrics
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		plugin.Metrics.CallCount++
		plugin.Metrics.TotalDuration += duration
		plugin.Metrics.AverageDuration = plugin.Metrics.TotalDuration / time.Duration(plugin.Metrics.CallCount)
		plugin.Metrics.LastCall = time.Now()
	}()

	result, err := plugin.Instance.ExecuteCommand(command, args)
	if err != nil {
		plugin.Metrics.ErrorCount++
		plugin.Metrics.LastError = time.Now()
		plugin.LastError = err.Error()
	}

	return result, err
}

// BroadcastEvent broadcasts an event to all active plugins
func (pm *PluginManager) BroadcastEvent(event *PluginEvent) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, plugin := range pm.plugins {
		if plugin.Status == StatusActive {
			go func(p *Plugin, n string) {
				if err := p.Instance.HandleEvent(event); err != nil {
					pm.logger.Error("Plugin %s error handling event: %v", n, err)
				}
			}(plugin, name)
		}
	}
}

// DiscoverPlugins discovers plugins in the plugin directory
func (pm *PluginManager) DiscoverPlugins() ([]string, error) {
	var plugins []string

	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			manifestPath := filepath.Join(pm.pluginDir, entry.Name(), "manifest.json")
			if _, err := os.Stat(manifestPath); err == nil {
				plugins = append(plugins, filepath.Join(pm.pluginDir, entry.Name()))
			}
		}
	}

	return plugins, nil
}

// EventBus handles event distribution
type EventBus struct {
	subscribers map[string][]func(*PluginEvent)
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(*PluginEvent)),
	}
}

// Subscribe subscribes to events
func (eb *EventBus) Subscribe(eventType string, handler func(*PluginEvent)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish publishes an event
func (eb *EventBus) Publish(event *PluginEvent) {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// PluginRegistry manages plugin metadata and discovery
type PluginRegistry struct {
	plugins map[string]*PluginManifest
	mu      sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]*PluginManifest),
	}
}

// Register registers a plugin in the registry
func (pr *PluginRegistry) Register(manifest *PluginManifest) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.plugins[manifest.Name] = manifest
}

// Unregister removes a plugin from the registry
func (pr *PluginRegistry) Unregister(name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	delete(pr.plugins, name)
}

// Get returns a plugin manifest
func (pr *PluginRegistry) Get(name string) (*PluginManifest, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	manifest, exists := pr.plugins[name]
	return manifest, exists
}

// List returns all registered plugins
func (pr *PluginRegistry) List() map[string]*PluginManifest {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugins := make(map[string]*PluginManifest)
	for name, manifest := range pr.plugins {
		plugins[name] = manifest
	}

	return plugins
}

// BasePlugin provides a base implementation for plugins
type BasePlugin struct {
	info   *PluginInfo
	config map[string]interface{}
	logger *logging.Logger
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(info *PluginInfo) *BasePlugin {
	return &BasePlugin{
		info:   info,
		config: make(map[string]interface{}),
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

// Initialize initializes the base plugin
func (bp *BasePlugin) Initialize(config map[string]interface{}) error {
	bp.config = config
	bp.logger.Info("Initialized plugin: %s", bp.info.Name)
	return nil
}

// Start starts the base plugin
func (bp *BasePlugin) Start(ctx context.Context) error {
	bp.logger.Info("Started plugin: %s", bp.info.Name)
	return nil
}

// Stop stops the base plugin
func (bp *BasePlugin) Stop() error {
	bp.logger.Info("Stopped plugin: %s", bp.info.Name)
	return nil
}

// GetInfo returns plugin info
func (bp *BasePlugin) GetInfo() *PluginInfo {
	return bp.info
}

// HandleEvent handles events (default implementation does nothing)
func (bp *BasePlugin) HandleEvent(event *PluginEvent) error {
	return nil
}

// GetCommands returns empty commands list
func (bp *BasePlugin) GetCommands() []*PluginCommand {
	return []*PluginCommand{}
}

// ExecuteCommand executes a command (default implementation returns error)
func (bp *BasePlugin) ExecuteCommand(command string, args map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("command %s not implemented", command)
}

// GetConfigSchema returns empty schema
func (bp *BasePlugin) GetConfigSchema() map[string]interface{} {
	return make(map[string]interface{})
}

// Validate validates configuration (default implementation accepts all)
func (bp *BasePlugin) Validate(config map[string]interface{}) error {
	return nil
}
