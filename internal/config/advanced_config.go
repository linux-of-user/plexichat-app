// Package config provides advanced configuration management
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"plexichat-client/internal/interfaces"
	"plexichat-client/pkg/logging"

	"gopkg.in/yaml.v3"
)

// AdvancedConfigManager implements sophisticated configuration management
type AdvancedConfigManager struct {
	mu             sync.RWMutex
	config         map[string]interface{}
	profiles       map[string]map[string]interface{}
	currentProfile string
	watchers       map[string][]func(interface{})
	validators     map[string]func(interface{}) error
	transformers   map[string]func(interface{}) interface{}
	sources        []ConfigSource
	logger         interfaces.Logger
	encryptionKey  []byte
	hotReload      bool
	reloadInterval time.Duration
	stopReload     chan struct{}
	defaults       map[string]interface{}
	schema         *ConfigSchema
}

// ConfigSource represents a configuration source
type ConfigSource interface {
	// Load loads configuration from the source
	Load(ctx context.Context) (map[string]interface{}, error)

	// Watch watches for configuration changes
	Watch(ctx context.Context, callback func(map[string]interface{})) error

	// GetPriority returns the source priority (higher = more important)
	GetPriority() int

	// GetName returns the source name
	GetName() string
}

// ConfigSchema defines the configuration schema for validation
type ConfigSchema struct {
	Fields map[string]FieldSchema `json:"fields"`
}

// FieldSchema defines validation rules for a configuration field
type FieldSchema struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Description string      `json:"description"`
}

// FileConfigSource loads configuration from files
type FileConfigSource struct {
	path     string
	format   string
	priority int
	watch    bool
}

// EnvironmentConfigSource loads configuration from environment variables
type EnvironmentConfigSource struct {
	prefix   string
	priority int
}

// RemoteConfigSource loads configuration from remote sources
type RemoteConfigSource struct {
	url      string
	headers  map[string]string
	priority int
	interval time.Duration
}

// NewAdvancedConfigManager creates a new advanced configuration manager
func NewAdvancedConfigManager() *AdvancedConfigManager {
	return &AdvancedConfigManager{
		config:         make(map[string]interface{}),
		profiles:       make(map[string]map[string]interface{}),
		currentProfile: "default",
		watchers:       make(map[string][]func(interface{})),
		validators:     make(map[string]func(interface{}) error),
		transformers:   make(map[string]func(interface{}) interface{}),
		sources:        make([]ConfigSource, 0),
		logger:         logging.GetLogger("config"),
		hotReload:      true,
		reloadInterval: 30 * time.Second,
		stopReload:     make(chan struct{}),
		defaults:       make(map[string]interface{}),
	}
}

// Load loads configuration from all sources
func (cm *AdvancedConfigManager) Load(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.logger.Info("Loading configuration from sources", "count", len(cm.sources))

	// Sort sources by priority
	cm.sortSources()

	// Load from all sources
	mergedConfig := make(map[string]interface{})

	// Start with defaults
	for key, value := range cm.defaults {
		mergedConfig[key] = value
	}

	// Load from sources in priority order
	for _, source := range cm.sources {
		sourceConfig, err := source.Load(ctx)
		if err != nil {
			cm.logger.Error("Failed to load from source", "source", source.GetName(), "error", err)
			continue
		}

		// Merge configuration
		cm.mergeConfig(mergedConfig, sourceConfig)
		cm.logger.Debug("Loaded configuration from source", "source", source.GetName())
	}

	// Apply transformations
	cm.applyTransformations(mergedConfig)

	// Validate configuration
	if err := cm.validateConfig(mergedConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Update current configuration
	cm.config = mergedConfig

	// Load profiles
	if err := cm.loadProfiles(ctx); err != nil {
		cm.logger.Error("Failed to load profiles", "error", err)
	}

	// Start hot reload if enabled
	if cm.hotReload {
		go cm.startHotReload(ctx)
	}

	cm.logger.Info("Configuration loaded successfully")
	return nil
}

// Get retrieves a configuration value
func (cm *AdvancedConfigManager) Get(key string) interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check current profile first
	if profile, exists := cm.profiles[cm.currentProfile]; exists {
		if value, exists := cm.getNestedValue(profile, key); exists {
			return value
		}
	}

	// Fall back to main configuration
	if value, exists := cm.getNestedValue(cm.config, key); exists {
		return value
	}

	return nil
}

// GetString retrieves a string configuration value
func (cm *AdvancedConfigManager) GetString(key string) string {
	value := cm.Get(key)
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInt retrieves an integer configuration value
func (cm *AdvancedConfigManager) GetInt(key string) int {
	value := cm.Get(key)
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	return 0
}

// GetBool retrieves a boolean configuration value
func (cm *AdvancedConfigManager) GetBool(key string) bool {
	value := cm.Get(key)
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1" || strings.ToLower(v) == "yes"
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	}

	return false
}

// GetDuration retrieves a duration configuration value
func (cm *AdvancedConfigManager) GetDuration(key string) time.Duration {
	value := cm.Get(key)
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	case int, int32, int64:
		return time.Duration(reflect.ValueOf(v).Int()) * time.Second
	case float32, float64:
		return time.Duration(reflect.ValueOf(v).Float()) * time.Second
	}

	return 0
}

// Set sets a configuration value
func (cm *AdvancedConfigManager) Set(key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate the value if validator exists
	if validator, exists := cm.validators[key]; exists {
		if err := validator(value); err != nil {
			return fmt.Errorf("validation failed for key '%s': %w", key, err)
		}
	}

	// Apply transformation if transformer exists
	if transformer, exists := cm.transformers[key]; exists {
		value = transformer(value)
	}

	// Set in current profile if it exists
	if profile, exists := cm.profiles[cm.currentProfile]; exists {
		cm.setNestedValue(profile, key, value)
	} else {
		// Set in main configuration
		cm.setNestedValue(cm.config, key, value)
	}

	// Notify watchers
	if watchers, exists := cm.watchers[key]; exists {
		for _, watcher := range watchers {
			go watcher(value)
		}
	}

	cm.logger.Debug("Configuration value set", "key", key, "value", value)
	return nil
}

// Watch watches for configuration changes
func (cm *AdvancedConfigManager) Watch(key string, callback func(interface{})) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.watchers[key] == nil {
		cm.watchers[key] = make([]func(interface{}), 0)
	}

	cm.watchers[key] = append(cm.watchers[key], callback)
	cm.logger.Debug("Added configuration watcher", "key", key)
	return nil
}

// Validate validates the current configuration
func (cm *AdvancedConfigManager) Validate() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.validateConfig(cm.config)
}

// Save saves configuration to persistent storage
func (cm *AdvancedConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Find writable file source
	for _, source := range cm.sources {
		if fileSource, ok := source.(*FileConfigSource); ok {
			return cm.saveToFile(fileSource.path, fileSource.format)
		}
	}

	return fmt.Errorf("no writable configuration source found")
}

// GetProfile returns the current configuration profile
func (cm *AdvancedConfigManager) GetProfile() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentProfile
}

// SetProfile sets the configuration profile
func (cm *AdvancedConfigManager) SetProfile(profile string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.profiles[profile]; !exists {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	oldProfile := cm.currentProfile
	cm.currentProfile = profile

	cm.logger.Info("Configuration profile changed", "from", oldProfile, "to", profile)
	return nil
}

// ListProfiles returns available configuration profiles
func (cm *AdvancedConfigManager) ListProfiles() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	profiles := make([]string, 0, len(cm.profiles))
	for profile := range cm.profiles {
		profiles = append(profiles, profile)
	}

	return profiles
}

// AddSource adds a configuration source
func (cm *AdvancedConfigManager) AddSource(source ConfigSource) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.sources = append(cm.sources, source)
	cm.logger.Debug("Added configuration source", "source", source.GetName(), "priority", source.GetPriority())
}

// SetValidator sets a validator for a configuration key
func (cm *AdvancedConfigManager) SetValidator(key string, validator func(interface{}) error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.validators[key] = validator
}

// SetTransformer sets a transformer for a configuration key
func (cm *AdvancedConfigManager) SetTransformer(key string, transformer func(interface{}) interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.transformers[key] = transformer
}

// SetDefault sets a default value for a configuration key
func (cm *AdvancedConfigManager) SetDefault(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.defaults[key] = value
}

// SetSchema sets the configuration schema for validation
func (cm *AdvancedConfigManager) SetSchema(schema *ConfigSchema) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.schema = schema
}

// Helper methods

// getNestedValue retrieves a nested configuration value using dot notation
func (cm *AdvancedConfigManager) getNestedValue(config map[string]interface{}, key string) (interface{}, bool) {
	keys := strings.Split(key, ".")
	current := config

	for i, k := range keys {
		if i == len(keys)-1 {
			value, exists := current[k]
			return value, exists
		}

		if next, exists := current[k]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return nil, false
}

// setNestedValue sets a nested configuration value using dot notation
func (cm *AdvancedConfigManager) setNestedValue(config map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	current := config

	for i, k := range keys {
		if i == len(keys)-1 {
			current[k] = value
			return
		}

		if next, exists := current[k]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Replace non-map value with map
				newMap := make(map[string]interface{})
				current[k] = newMap
				current = newMap
			}
		} else {
			// Create new map
			newMap := make(map[string]interface{})
			current[k] = newMap
			current = newMap
		}
	}
}

// mergeConfig merges source configuration into target
func (cm *AdvancedConfigManager) mergeConfig(target, source map[string]interface{}) {
	for key, value := range source {
		if targetValue, exists := target[key]; exists {
			if targetMap, ok := targetValue.(map[string]interface{}); ok {
				if sourceMap, ok := value.(map[string]interface{}); ok {
					cm.mergeConfig(targetMap, sourceMap)
					continue
				}
			}
		}
		target[key] = value
	}
}

// applyTransformations applies all registered transformations
func (cm *AdvancedConfigManager) applyTransformations(config map[string]interface{}) {
	for key, transformer := range cm.transformers {
		if value, exists := cm.getNestedValue(config, key); exists {
			transformed := transformer(value)
			cm.setNestedValue(config, key, transformed)
		}
	}
}

// validateConfig validates configuration against schema and validators
func (cm *AdvancedConfigManager) validateConfig(config map[string]interface{}) error {
	// Validate against schema if available
	if cm.schema != nil {
		if err := cm.validateAgainstSchema(config); err != nil {
			return err
		}
	}

	// Apply custom validators
	for key, validator := range cm.validators {
		if value, exists := cm.getNestedValue(config, key); exists {
			if err := validator(value); err != nil {
				return fmt.Errorf("validation failed for key '%s': %w", key, err)
			}
		}
	}

	return nil
}

// validateAgainstSchema validates configuration against the schema
func (cm *AdvancedConfigManager) validateAgainstSchema(config map[string]interface{}) error {
	for fieldName, fieldSchema := range cm.schema.Fields {
		value, exists := cm.getNestedValue(config, fieldName)

		// Check required fields
		if fieldSchema.Required && !exists {
			return fmt.Errorf("required field '%s' is missing", fieldName)
		}

		if !exists {
			continue
		}

		// Validate type
		if err := cm.validateFieldType(fieldName, value, fieldSchema); err != nil {
			return err
		}

		// Validate constraints
		if err := cm.validateFieldConstraints(fieldName, value, fieldSchema); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldType validates field type
func (cm *AdvancedConfigManager) validateFieldType(fieldName string, value interface{}, schema FieldSchema) error {
	valueType := reflect.TypeOf(value).Kind().String()

	switch schema.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %s", fieldName, valueType)
		}
	case "int":
		switch value.(type) {
		case int, int32, int64:
		default:
			return fmt.Errorf("field '%s' must be an integer, got %s", fieldName, valueType)
		}
	case "float":
		switch value.(type) {
		case float32, float64:
		default:
			return fmt.Errorf("field '%s' must be a float, got %s", fieldName, valueType)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %s", fieldName, valueType)
		}
	case "duration":
		switch value.(type) {
		case time.Duration, string:
		default:
			return fmt.Errorf("field '%s' must be a duration, got %s", fieldName, valueType)
		}
	}

	return nil
}

// validateFieldConstraints validates field constraints
func (cm *AdvancedConfigManager) validateFieldConstraints(fieldName string, value interface{}, schema FieldSchema) error {
	// Validate enum
	if len(schema.Enum) > 0 {
		valueStr := fmt.Sprintf("%v", value)
		found := false
		for _, enumValue := range schema.Enum {
			if enumValue == valueStr {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field '%s' must be one of %v, got %v", fieldName, schema.Enum, value)
		}
	}

	// Validate min/max for numeric types
	if schema.Min != nil || schema.Max != nil {
		if err := cm.validateNumericRange(fieldName, value, schema.Min, schema.Max); err != nil {
			return err
		}
	}

	return nil
}

// validateNumericRange validates numeric range constraints
func (cm *AdvancedConfigManager) validateNumericRange(fieldName string, value, min, max interface{}) error {
	var numValue float64
	var ok bool

	switch v := value.(type) {
	case int:
		numValue = float64(v)
		ok = true
	case int32:
		numValue = float64(v)
		ok = true
	case int64:
		numValue = float64(v)
		ok = true
	case float32:
		numValue = float64(v)
		ok = true
	case float64:
		numValue = v
		ok = true
	}

	if !ok {
		return nil // Not a numeric type
	}

	if min != nil {
		if minFloat, ok := min.(float64); ok && numValue < minFloat {
			return fmt.Errorf("field '%s' must be >= %v, got %v", fieldName, min, value)
		}
	}

	if max != nil {
		if maxFloat, ok := max.(float64); ok && numValue > maxFloat {
			return fmt.Errorf("field '%s' must be <= %v, got %v", fieldName, max, value)
		}
	}

	return nil
}

// sortSources sorts configuration sources by priority
func (cm *AdvancedConfigManager) sortSources() {
	for i := 0; i < len(cm.sources); i++ {
		for j := i + 1; j < len(cm.sources); j++ {
			if cm.sources[i].GetPriority() < cm.sources[j].GetPriority() {
				cm.sources[i], cm.sources[j] = cm.sources[j], cm.sources[i]
			}
		}
	}
}

// loadProfiles loads configuration profiles
func (cm *AdvancedConfigManager) loadProfiles(ctx context.Context) error {
	// Look for profile-specific configuration files
	for _, source := range cm.sources {
		if fileSource, ok := source.(*FileConfigSource); ok {
			profilesDir := filepath.Dir(fileSource.path) + "/profiles"
			if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
				continue
			}

			files, err := os.ReadDir(profilesDir)
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				profileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
				profilePath := filepath.Join(profilesDir, file.Name())

				profileConfig, err := cm.loadFromFile(profilePath, fileSource.format)
				if err != nil {
					cm.logger.Error("Failed to load profile", "profile", profileName, "error", err)
					continue
				}

				cm.profiles[profileName] = profileConfig
				cm.logger.Debug("Loaded configuration profile", "profile", profileName)
			}
		}
	}

	return nil
}

// startHotReload starts the hot reload mechanism
func (cm *AdvancedConfigManager) startHotReload(ctx context.Context) {
	ticker := time.NewTicker(cm.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cm.Load(ctx); err != nil {
				cm.logger.Error("Hot reload failed", "error", err)
			}
		case <-cm.stopReload:
			return
		case <-ctx.Done():
			return
		}
	}
}

// saveToFile saves configuration to a file
func (cm *AdvancedConfigManager) saveToFile(path, format string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cm.config)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Encrypt if encryption key is set
	if cm.encryptionKey != nil {
		data, err = cm.encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt configuration: %w", err)
		}
	}

	return os.WriteFile(path, data, 0600)
}

// loadFromFile loads configuration from a file
func (cm *AdvancedConfigManager) loadFromFile(path, format string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decrypt if encryption key is set
	if cm.encryptionKey != nil {
		data, err = cm.decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt configuration: %w", err)
		}
	}

	var config map[string]interface{}

	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return config, nil
}

// encrypt encrypts data using the encryption key
func (cm *AdvancedConfigManager) encrypt(data []byte) ([]byte, error) {
	// TODO: Implement encryption (AES-GCM recommended)
	return data, nil
}

// decrypt decrypts data using the encryption key
func (cm *AdvancedConfigManager) decrypt(data []byte) ([]byte, error) {
	// TODO: Implement decryption (AES-GCM recommended)
	return data, nil
}

// Stop stops the configuration manager
func (cm *AdvancedConfigManager) Stop() {
	close(cm.stopReload)
}

// Configuration Source Implementations

// NewFileConfigSource creates a new file configuration source
func NewFileConfigSource(path, format string, priority int, watch bool) *FileConfigSource {
	return &FileConfigSource{
		path:     path,
		format:   format,
		priority: priority,
		watch:    watch,
	}
}

// Load loads configuration from file
func (fs *FileConfigSource) Load(ctx context.Context) (map[string]interface{}, error) {
	data, err := os.ReadFile(fs.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", fs.path, err)
	}

	var config map[string]interface{}

	switch strings.ToLower(fs.format) {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", fs.format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return config, nil
}

// Watch watches for file changes
func (fs *FileConfigSource) Watch(ctx context.Context, callback func(map[string]interface{})) error {
	if !fs.watch {
		return nil
	}

	// TODO: Implement file watching using fsnotify
	return nil
}

// GetPriority returns the source priority
func (fs *FileConfigSource) GetPriority() int {
	return fs.priority
}

// GetName returns the source name
func (fs *FileConfigSource) GetName() string {
	return fmt.Sprintf("file:%s", fs.path)
}
