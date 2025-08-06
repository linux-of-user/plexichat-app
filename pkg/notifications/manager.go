package notifications

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/config"
	"plexichat-client/pkg/events"
	"plexichat-client/pkg/logging"
)

// NotificationManager manages notifications
type NotificationManager struct {
	config    *config.NotificationConfig
	logger    *logging.Logger
	providers map[string]NotificationProvider
	rules     []*NotificationRule
	eventBus  *events.EventBus
	mu        sync.RWMutex
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NotificationProvider defines the interface for notification providers
type NotificationProvider interface {
	Send(ctx context.Context, notification *Notification) error
	GetType() string
	IsEnabled() bool
	Configure(config map[string]interface{}) error
}

// Notification represents a notification
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Icon        string                 `json:"icon,omitempty"`
	Sound       string                 `json:"sound,omitempty"`
	Priority    NotificationPriority   `json:"priority"`
	Category    string                 `json:"category"`
	ChannelID   string                 `json:"channel_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Actions     []NotificationAction   `json:"actions,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Delivered   bool                   `json:"delivered"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeMessage     NotificationType = "message"
	NotificationTypeMention     NotificationType = "mention"
	NotificationTypeReaction    NotificationType = "reaction"
	NotificationTypeChannelJoin NotificationType = "channel_join"
	NotificationTypeUserOnline  NotificationType = "user_online"
	NotificationTypeFileUpload  NotificationType = "file_upload"
	NotificationTypeSystem      NotificationType = "system"
	NotificationTypeError       NotificationType = "error"
	NotificationTypeWarning     NotificationType = "warning"
	NotificationTypeInfo        NotificationType = "info"
)

// NotificationPriority represents the priority of a notification
type NotificationPriority int

const (
	PriorityLow NotificationPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// NotificationAction represents an action that can be taken on a notification
type NotificationAction struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NotificationRule represents a rule for filtering notifications
type NotificationRule struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Enabled    bool                    `json:"enabled"`
	Conditions []NotificationCondition `json:"conditions"`
	Actions    []string                `json:"actions"`
	Priority   NotificationPriority    `json:"priority"`
	Sound      string                  `json:"sound,omitempty"`
	Providers  []string                `json:"providers"`
	QuietHours *QuietHours             `json:"quiet_hours,omitempty"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

// NotificationCondition represents a condition for notification rules
type NotificationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// QuietHours represents quiet hours configuration
type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Timezone  string `json:"timezone"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config *config.NotificationConfig, eventBus *events.EventBus) *NotificationManager {
	ctx, cancel := context.WithCancel(context.Background())

	nm := &NotificationManager{
		config:    config,
		logger:    logging.NewLogger(logging.INFO, nil, true),
		providers: make(map[string]NotificationProvider),
		rules:     make([]*NotificationRule, 0),
		eventBus:  eventBus,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register built-in providers
	nm.registerBuiltinProviders()

	// Load default rules
	nm.loadDefaultRules()

	return nm
}

// Start starts the notification manager
func (nm *NotificationManager) Start() error {
	nm.mu.Lock()
	if nm.running {
		nm.mu.Unlock()
		return fmt.Errorf("notification manager already running")
	}
	nm.running = true
	nm.mu.Unlock()

	// Subscribe to events
	if nm.eventBus != nil {
		nm.eventBus.Subscribe(&NotificationEventHandler{manager: nm})
	}

	nm.logger.Info("Notification manager started")
	return nil
}

// Stop stops the notification manager
func (nm *NotificationManager) Stop() error {
	nm.mu.Lock()
	if !nm.running {
		nm.mu.Unlock()
		return fmt.Errorf("notification manager not running")
	}
	nm.running = false
	nm.mu.Unlock()

	nm.cancel()
	nm.logger.Info("Notification manager stopped")
	return nil
}

// Send sends a notification
func (nm *NotificationManager) Send(ctx context.Context, notification *Notification) error {
	if !nm.config.Enabled {
		return nil
	}

	// Set defaults
	if notification.ID == "" {
		notification.ID = nm.generateNotificationID()
	}
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}
	if notification.Priority == 0 {
		notification.Priority = PriorityNormal
	}

	// Check quiet hours
	if nm.isQuietHours() && notification.Priority < PriorityCritical {
		nm.logger.Debug("Skipping notification due to quiet hours: %s", notification.ID)
		return nil
	}

	// Apply rules
	providers := nm.getProvidersForNotification(notification)
	if len(providers) == 0 {
		nm.logger.Debug("No providers enabled for notification: %s", notification.ID)
		return nil
	}

	// Send to providers
	var lastError error
	for _, providerName := range providers {
		provider, exists := nm.providers[providerName]
		if !exists || !provider.IsEnabled() {
			continue
		}

		if err := provider.Send(ctx, notification); err != nil {
			nm.logger.Error("Failed to send notification via %s: %v", providerName, err)
			lastError = err
		} else {
			nm.logger.Debug("Notification sent via %s: %s", providerName, notification.ID)
		}
	}

	// Mark as delivered if at least one provider succeeded
	if lastError == nil {
		notification.Delivered = true
		now := time.Now()
		notification.DeliveredAt = &now
	}

	return lastError
}

// AddRule adds a notification rule
func (nm *NotificationManager) AddRule(rule *NotificationRule) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if rule.ID == "" {
		rule.ID = nm.generateRuleID()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	nm.rules = append(nm.rules, rule)
	nm.logger.Info("Added notification rule: %s", rule.Name)
}

// RemoveRule removes a notification rule
func (nm *NotificationManager) RemoveRule(ruleID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i, rule := range nm.rules {
		if rule.ID == ruleID {
			nm.rules = append(nm.rules[:i], nm.rules[i+1:]...)
			nm.logger.Info("Removed notification rule: %s", rule.Name)
			return
		}
	}
}

// GetRules returns all notification rules
func (nm *NotificationManager) GetRules() []*NotificationRule {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	rules := make([]*NotificationRule, len(nm.rules))
	copy(rules, nm.rules)
	return rules
}

// RegisterProvider registers a notification provider
func (nm *NotificationManager) RegisterProvider(provider NotificationProvider) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.providers[provider.GetType()] = provider
	nm.logger.Info("Registered notification provider: %s", provider.GetType())
}

// registerBuiltinProviders registers built-in notification providers
func (nm *NotificationManager) registerBuiltinProviders() {
	// Desktop notifications
	if nm.config.Desktop {
		nm.RegisterProvider(NewDesktopProvider())
	}

	// Sound notifications
	if nm.config.Sound {
		nm.RegisterProvider(NewSoundProvider())
	}

	// Email notifications
	if nm.config.Email && nm.config.EmailSettings.SMTPHost != "" {
		provider := NewEmailProvider()
		provider.Configure(map[string]interface{}{
			"smtp_host":     nm.config.EmailSettings.SMTPHost,
			"smtp_port":     nm.config.EmailSettings.SMTPPort,
			"smtp_username": nm.config.EmailSettings.SMTPUsername,
			"smtp_password": nm.config.EmailSettings.SMTPPassword,
			"from_address":  nm.config.EmailSettings.FromAddress,
			"tls_enabled":   nm.config.EmailSettings.TLSEnabled,
		})
		nm.RegisterProvider(provider)
	}

	// Push notifications
	if nm.config.Push && nm.config.PushSettings.ServiceURL != "" {
		provider := NewPushProvider()
		provider.Configure(map[string]interface{}{
			"service_url": nm.config.PushSettings.ServiceURL,
			"api_key":     nm.config.PushSettings.APIKey,
			"device_id":   nm.config.PushSettings.DeviceID,
			"headers":     nm.config.PushSettings.Headers,
		})
		nm.RegisterProvider(provider)
	}
}

// loadDefaultRules loads default notification rules
func (nm *NotificationManager) loadDefaultRules() {
	// Mention notifications
	nm.AddRule(&NotificationRule{
		Name:    "Mentions",
		Enabled: true,
		Conditions: []NotificationCondition{
			{Field: "type", Operator: "equals", Value: string(NotificationTypeMention)},
		},
		Actions:   []string{"desktop", "sound"},
		Priority:  PriorityHigh,
		Sound:     "mention.wav",
		Providers: []string{"desktop", "sound"},
	})

	// Direct messages
	nm.AddRule(&NotificationRule{
		Name:    "Direct Messages",
		Enabled: true,
		Conditions: []NotificationCondition{
			{Field: "type", Operator: "equals", Value: string(NotificationTypeMessage)},
			{Field: "category", Operator: "equals", Value: "direct"},
		},
		Actions:   []string{"desktop", "sound"},
		Priority:  PriorityHigh,
		Sound:     "message.wav",
		Providers: []string{"desktop", "sound"},
	})

	// System notifications
	nm.AddRule(&NotificationRule{
		Name:    "System Notifications",
		Enabled: true,
		Conditions: []NotificationCondition{
			{Field: "type", Operator: "equals", Value: string(NotificationTypeSystem)},
		},
		Actions:   []string{"desktop"},
		Priority:  PriorityNormal,
		Providers: []string{"desktop"},
	})
}

// Helper methods

func (nm *NotificationManager) generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

func (nm *NotificationManager) generateRuleID() string {
	return fmt.Sprintf("rule_%d", time.Now().UnixNano())
}

func (nm *NotificationManager) isQuietHours() bool {
	if !nm.config.QuietHours {
		return false
	}

	now := time.Now()
	startTime, err := time.Parse("15:04", nm.config.QuietStart)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04", nm.config.QuietEnd)
	if err != nil {
		return false
	}

	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	start := time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	end := time.Date(0, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	if start.Before(end) {
		return currentTime.After(start) && currentTime.Before(end)
	} else {
		return currentTime.After(start) || currentTime.Before(end)
	}
}

func (nm *NotificationManager) getProvidersForNotification(notification *Notification) []string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	providers := make([]string, 0)

	for _, rule := range nm.rules {
		if !rule.Enabled {
			continue
		}

		if nm.matchesRule(notification, rule) {
			providers = append(providers, rule.Providers...)
		}
	}

	// Remove duplicates
	uniqueProviders := make([]string, 0)
	seen := make(map[string]bool)
	for _, provider := range providers {
		if !seen[provider] {
			uniqueProviders = append(uniqueProviders, provider)
			seen[provider] = true
		}
	}

	return uniqueProviders
}

func (nm *NotificationManager) matchesRule(notification *Notification, rule *NotificationRule) bool {
	for _, condition := range rule.Conditions {
		if !nm.matchesCondition(notification, condition) {
			return false
		}
	}
	return true
}

func (nm *NotificationManager) matchesCondition(notification *Notification, condition NotificationCondition) bool {
	var fieldValue interface{}

	switch condition.Field {
	case "type":
		fieldValue = string(notification.Type)
	case "priority":
		fieldValue = int(notification.Priority)
	case "category":
		fieldValue = notification.Category
	case "channel_id":
		fieldValue = notification.ChannelID
	case "user_id":
		fieldValue = notification.UserID
	default:
		if notification.Data != nil {
			fieldValue = notification.Data[condition.Field]
		}
	}

	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return strings.Contains(str, substr)
			}
		}
	case "greater_than":
		if num, ok := fieldValue.(int); ok {
			if target, ok := condition.Value.(int); ok {
				return num > target
			}
		}
	case "less_than":
		if num, ok := fieldValue.(int); ok {
			if target, ok := condition.Value.(int); ok {
				return num < target
			}
		}
	}

	return false
}

// NotificationEventHandler handles events and creates notifications
type NotificationEventHandler struct {
	manager *NotificationManager
}

func (neh *NotificationEventHandler) Handle(ctx context.Context, event *events.Event) error {
	var notification *Notification

	switch event.Type {
	case "message.received":
		notification = neh.createMessageNotification(event)
	case "user.mentioned":
		notification = neh.createMentionNotification(event)
	case "system.error":
		notification = neh.createSystemNotification(event)
	default:
		return nil // No notification needed
	}

	if notification != nil {
		return neh.manager.Send(ctx, notification)
	}

	return nil
}

func (neh *NotificationEventHandler) GetEventTypes() []string {
	return []string{"message.received", "user.mentioned", "system.error", "user.online", "file.uploaded"}
}

func (neh *NotificationEventHandler) GetPriority() int {
	return 50
}

func (neh *NotificationEventHandler) createMessageNotification(event *events.Event) *Notification {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &Notification{
		Type:      NotificationTypeMessage,
		Title:     "New Message",
		Message:   fmt.Sprintf("From %s: %s", data["username"], data["content"]),
		Priority:  PriorityNormal,
		Category:  "message",
		ChannelID: fmt.Sprintf("%v", data["channel_id"]),
		UserID:    fmt.Sprintf("%v", data["user_id"]),
		Data:      data,
	}
}

func (neh *NotificationEventHandler) createMentionNotification(event *events.Event) *Notification {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &Notification{
		Type:     NotificationTypeMention,
		Title:    "You were mentioned",
		Message:  fmt.Sprintf("%s mentioned you: %s", data["username"], data["content"]),
		Priority: PriorityHigh,
		Category: "mention",
		Sound:    "mention.wav",
		Data:     data,
	}
}

func (neh *NotificationEventHandler) createSystemNotification(event *events.Event) *Notification {
	return &Notification{
		Type:     NotificationTypeSystem,
		Title:    "System Notification",
		Message:  fmt.Sprintf("%v", event.Data),
		Priority: PriorityNormal,
		Category: "system",
	}
}
