package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotificationMessage     NotificationType = "message"
	NotificationMention     NotificationType = "mention"
	NotificationReply       NotificationType = "reply"
	NotificationFileUpload  NotificationType = "file_upload"
	NotificationSystemAlert NotificationType = "system_alert"
	NotificationError       NotificationType = "error"
	NotificationWarning     NotificationType = "warning"
	NotificationInfo        NotificationType = "info"
	NotificationSuccess     NotificationType = "success"
	NotificationUpdate      NotificationType = "update"
	NotificationReminder    NotificationType = "reminder"
	NotificationCall        NotificationType = "call"
	NotificationInvite      NotificationType = "invite"
)

// NotificationPriority represents notification priority levels
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
	PriorityUrgent   NotificationPriority = "urgent"
)

// NotificationStatus represents notification status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusDelivered NotificationStatus = "delivered"
	StatusRead      NotificationStatus = "read"
	StatusDismissed NotificationStatus = "dismissed"
	StatusFailed    NotificationStatus = "failed"
	StatusExpired   NotificationStatus = "expired"
)

// Notification represents a notification
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Priority    NotificationPriority   `json:"priority"`
	Status      NotificationStatus     `json:"status"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Icon        string                 `json:"icon,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Sound       string                 `json:"sound,omitempty"`
	Badge       int                    `json:"badge,omitempty"`
	Actions     []*NotificationAction  `json:"actions,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	ChannelID   string                 `json:"channel_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	DismissedAt *time.Time             `json:"dismissed_at,omitempty"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	FailedAt    *time.Time             `json:"failed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Tags        []string               `json:"tags,omitempty"`
	Category    string                 `json:"category,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
	GroupID     string                 `json:"group_id,omitempty"`
}

// NotificationAction represents an action that can be taken on a notification
type NotificationAction struct {
	ID          string                                         `json:"id"`
	Title       string                                         `json:"title"`
	Icon        string                                         `json:"icon,omitempty"`
	Type        string                                         `json:"type"` // button, input, select
	Destructive bool                                           `json:"destructive,omitempty"`
	Data        map[string]interface{}                         `json:"data,omitempty"`
	Handler     func(*Notification, *NotificationAction) error `json:"-"`
}

// NotificationChannel represents a delivery channel for notifications
type NotificationChannel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // desktop, push, email, sms, webhook
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	Filters     []*NotificationFilter  `json:"filters"`
	RateLimit   *RateLimit             `json:"rate_limit,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
}

// NotificationFilter represents a filter for notifications
type NotificationFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, contains, regex, in, not_in
	Value    interface{} `json:"value"`
	Negate   bool        `json:"negate,omitempty"`
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	MaxNotifications int           `json:"max_notifications"`
	TimeWindow       time.Duration `json:"time_window"`
	BurstSize        int           `json:"burst_size,omitempty"`
}

// RetryPolicy represents retry configuration
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// NotificationConfig represents notification system configuration
type NotificationConfig struct {
	Enabled          bool                            `json:"enabled"`
	DefaultChannel   string                          `json:"default_channel"`
	Channels         map[string]*NotificationChannel `json:"channels"`
	GlobalFilters    []*NotificationFilter           `json:"global_filters"`
	QuietHours       *QuietHours                     `json:"quiet_hours,omitempty"`
	DoNotDisturb     bool                            `json:"do_not_disturb"`
	BadgeCount       bool                            `json:"badge_count"`
	Sounds           bool                            `json:"sounds"`
	Vibration        bool                            `json:"vibration"`
	StorageDir       string                          `json:"storage_dir"`
	RetentionDays    int                             `json:"retention_days"`
	MaxNotifications int                             `json:"max_notifications"`
	GroupSimilar     bool                            `json:"group_similar"`
	GroupTimeWindow  time.Duration                   `json:"group_time_window"`
}

// QuietHours represents quiet hours configuration
type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	Timezone  string `json:"timezone"`
	Weekdays  []int  `json:"weekdays"` // 0=Sunday, 1=Monday, etc.
}

// NotificationManager manages the notification system
type NotificationManager struct {
	config        *NotificationConfig
	notifications map[string]*Notification
	channels      map[string]NotificationChannel
	handlers      map[string]NotificationHandler
	storage       *NotificationStorage
	logger        *logging.Logger
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	eventChan     chan *NotificationEvent
	badgeCount    int
}

// NotificationHandler interface for handling notifications
type NotificationHandler interface {
	CanHandle(notification *Notification) bool
	Handle(ctx context.Context, notification *Notification) error
	GetChannelType() string
}

// NotificationEvent represents notification events
type NotificationEvent struct {
	Type         string                 `json:"type"`
	Notification *Notification          `json:"notification"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config *NotificationConfig) *NotificationManager {
	if config == nil {
		config = &NotificationConfig{
			Enabled:          true,
			DefaultChannel:   "desktop",
			Channels:         make(map[string]*NotificationChannel),
			GlobalFilters:    make([]*NotificationFilter, 0),
			DoNotDisturb:     false,
			BadgeCount:       true,
			Sounds:           true,
			Vibration:        true,
			StorageDir:       "notifications",
			RetentionDays:    30,
			MaxNotifications: 1000,
			GroupSimilar:     true,
			GroupTimeWindow:  5 * time.Minute,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	nm := &NotificationManager{
		config:        config,
		notifications: make(map[string]*Notification),
		channels:      make(map[string]NotificationChannel),
		handlers:      make(map[string]NotificationHandler),
		storage:       NewNotificationStorage(config.StorageDir),
		logger:        logging.NewLogger(logging.INFO, nil, true),
		ctx:           ctx,
		cancel:        cancel,
		eventChan:     make(chan *NotificationEvent, 1000),
	}

	// Initialize default channels
	nm.initializeDefaultChannels()

	// Start background processing
	if config.Enabled {
		go nm.processEvents()
		go nm.cleanupExpired()
	}

	return nm
}

// Send sends a notification
func (nm *NotificationManager) Send(notification *Notification) error {
	if !nm.config.Enabled {
		return fmt.Errorf("notifications are disabled")
	}

	// Set defaults
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	if notification.Priority == "" {
		notification.Priority = PriorityNormal
	}
	if notification.Status == "" {
		notification.Status = StatusPending
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	notification.UpdatedAt = time.Now()

	// Apply global filters
	if !nm.passesFilters(notification, nm.config.GlobalFilters) {
		nm.logger.Debug("Notification filtered out by global filters: %s", notification.ID)
		return nil
	}

	// Check quiet hours
	if nm.isQuietHours() && notification.Priority != PriorityCritical && notification.Priority != PriorityUrgent {
		nm.logger.Debug("Notification delayed due to quiet hours: %s", notification.ID)
		// Schedule for later delivery
		return nm.scheduleNotification(notification)
	}

	// Check do not disturb
	if nm.config.DoNotDisturb && notification.Priority != PriorityCritical && notification.Priority != PriorityUrgent {
		nm.logger.Debug("Notification suppressed due to do not disturb: %s", notification.ID)
		return nil
	}

	// Store notification
	nm.mu.Lock()
	nm.notifications[notification.ID] = notification
	nm.mu.Unlock()

	// Save to persistent storage
	if err := nm.storage.StoreNotification(notification); err != nil {
		nm.logger.Error("Failed to store notification: %v", err)
	}

	// Send to event channel for processing
	event := &NotificationEvent{
		Type:         "notification_created",
		Notification: notification,
		Timestamp:    time.Now(),
	}

	select {
	case nm.eventChan <- event:
	default:
		nm.logger.Error("Notification event queue full, dropping event")
	}

	nm.logger.Info("Notification sent: %s (%s)", notification.Title, notification.ID)
	return nil
}

// SendSimple sends a simple notification with just title and message
func (nm *NotificationManager) SendSimple(notificationType NotificationType, title, message string) error {
	notification := &Notification{
		Type:    notificationType,
		Title:   title,
		Message: message,
	}
	return nm.Send(notification)
}

// SendWithActions sends a notification with actions
func (nm *NotificationManager) SendWithActions(notificationType NotificationType, title, message string, actions []*NotificationAction) error {
	notification := &Notification{
		Type:    notificationType,
		Title:   title,
		Message: message,
		Actions: actions,
	}
	return nm.Send(notification)
}

// MarkAsRead marks a notification as read
func (nm *NotificationManager) MarkAsRead(notificationID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification, exists := nm.notifications[notificationID]
	if !exists {
		return fmt.Errorf("notification not found: %s", notificationID)
	}

	if notification.Status != StatusRead {
		notification.Status = StatusRead
		now := time.Now()
		notification.ReadAt = &now
		notification.UpdatedAt = now

		// Update badge count
		if nm.config.BadgeCount {
			nm.updateBadgeCount()
		}

		// Save to storage
		if err := nm.storage.UpdateNotification(notification); err != nil {
			nm.logger.Error("Failed to update notification: %v", err)
		}

		// Send event
		event := &NotificationEvent{
			Type:         "notification_read",
			Notification: notification,
			Timestamp:    time.Now(),
		}

		select {
		case nm.eventChan <- event:
		default:
		}

		nm.logger.Debug("Notification marked as read: %s", notificationID)
	}

	return nil
}

// MarkAsDismissed marks a notification as dismissed
func (nm *NotificationManager) MarkAsDismissed(notificationID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification, exists := nm.notifications[notificationID]
	if !exists {
		return fmt.Errorf("notification not found: %s", notificationID)
	}

	if notification.Status != StatusDismissed {
		notification.Status = StatusDismissed
		now := time.Now()
		notification.DismissedAt = &now
		notification.UpdatedAt = now

		// Update badge count
		if nm.config.BadgeCount {
			nm.updateBadgeCount()
		}

		// Save to storage
		if err := nm.storage.UpdateNotification(notification); err != nil {
			nm.logger.Error("Failed to update notification: %v", err)
		}

		// Send event
		event := &NotificationEvent{
			Type:         "notification_dismissed",
			Notification: notification,
			Timestamp:    time.Now(),
		}

		select {
		case nm.eventChan <- event:
		default:
		}

		nm.logger.Debug("Notification dismissed: %s", notificationID)
	}

	return nil
}

// GetNotification retrieves a notification by ID
func (nm *NotificationManager) GetNotification(notificationID string) (*Notification, bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	notification, exists := nm.notifications[notificationID]
	return notification, exists
}

// GetNotifications retrieves notifications with optional filters
func (nm *NotificationManager) GetNotifications(filters map[string]interface{}) []*Notification {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var result []*Notification

	for _, notification := range nm.notifications {
		if nm.matchesFilters(notification, filters) {
			result = append(result, notification)
		}
	}

	return result
}

// GetUnreadCount returns the count of unread notifications
func (nm *NotificationManager) GetUnreadCount() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	count := 0
	for _, notification := range nm.notifications {
		if notification.Status == StatusPending || notification.Status == StatusDelivered {
			count++
		}
	}

	return count
}

// GetBadgeCount returns the current badge count
func (nm *NotificationManager) GetBadgeCount() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.badgeCount
}

// ClearAll clears all notifications
func (nm *NotificationManager) ClearAll() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, notification := range nm.notifications {
		notification.Status = StatusDismissed
		now := time.Now()
		notification.DismissedAt = &now
		notification.UpdatedAt = now

		// Save to storage
		if err := nm.storage.UpdateNotification(notification); err != nil {
			nm.logger.Error("Failed to update notification: %v", err)
		}
	}

	// Reset badge count
	nm.badgeCount = 0

	nm.logger.Info("All notifications cleared")
	return nil
}

// RegisterHandler registers a notification handler
func (nm *NotificationManager) RegisterHandler(handler NotificationHandler) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.handlers[handler.GetChannelType()] = handler
	nm.logger.Info("Registered notification handler: %s", handler.GetChannelType())
}

// UnregisterHandler unregisters a notification handler
func (nm *NotificationManager) UnregisterHandler(channelType string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.handlers, channelType)
	nm.logger.Info("Unregistered notification handler: %s", channelType)
}

// UpdateConfig updates the notification configuration
func (nm *NotificationManager) UpdateConfig(config *NotificationConfig) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.config = config
	nm.logger.Info("Notification configuration updated")
}

// GetConfig returns the current configuration
func (nm *NotificationManager) GetConfig() *NotificationConfig {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.config
}

// GetStats returns notification statistics
func (nm *NotificationManager) GetStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_notifications": len(nm.notifications),
		"unread_count":        nm.GetUnreadCount(),
		"badge_count":         nm.badgeCount,
	}

	// Count by status
	statusCounts := make(map[NotificationStatus]int)
	typeCounts := make(map[NotificationType]int)
	priorityCounts := make(map[NotificationPriority]int)

	for _, notification := range nm.notifications {
		statusCounts[notification.Status]++
		typeCounts[notification.Type]++
		priorityCounts[notification.Priority]++
	}

	stats["by_status"] = statusCounts
	stats["by_type"] = typeCounts
	stats["by_priority"] = priorityCounts

	return stats
}

// processEvents processes notification events in background
func (nm *NotificationManager) processEvents() {
	for {
		select {
		case event := <-nm.eventChan:
			nm.handleEvent(event)
		case <-nm.ctx.Done():
			return
		}
	}
}

// handleEvent handles a notification event
func (nm *NotificationManager) handleEvent(event *NotificationEvent) {
	switch event.Type {
	case "notification_created":
		nm.deliverNotification(event.Notification)
	case "notification_read":
		// Handle read event
	case "notification_dismissed":
		// Handle dismissed event
	}
}

// deliverNotification delivers a notification through appropriate channels
func (nm *NotificationManager) deliverNotification(notification *Notification) {
	// Determine which channels to use
	channels := nm.getChannelsForNotification(notification)

	for _, channelID := range channels {
		channel, exists := nm.config.Channels[channelID]
		if !exists || !channel.Enabled {
			continue
		}

		// Check channel filters
		if !nm.passesFilters(notification, channel.Filters) {
			continue
		}

		// Check rate limits
		if channel.RateLimit != nil && nm.isRateLimited(channelID, channel.RateLimit) {
			nm.logger.Debug("Notification rate limited for channel %s", channelID)
			continue
		}

		// Find handler for channel type
		handler, exists := nm.handlers[channel.Type]
		if !exists {
			nm.logger.Error("No handler found for channel type: %s", channel.Type)
			continue
		}

		// Deliver notification
		go func(h NotificationHandler, n *Notification, ch *NotificationChannel) {
			ctx, cancel := context.WithTimeout(nm.ctx, 30*time.Second)
			defer cancel()

			if err := h.Handle(ctx, n); err != nil {
				nm.logger.Error("Failed to deliver notification via %s: %v", ch.Type, err)
				nm.handleDeliveryFailure(n, ch, err)
			} else {
				nm.handleDeliverySuccess(n, ch)
			}
		}(handler, notification, channel)
	}
}

// Helper methods
func (nm *NotificationManager) initializeDefaultChannels() {
	// Desktop notifications
	nm.config.Channels["desktop"] = &NotificationChannel{
		ID:      "desktop",
		Name:    "Desktop Notifications",
		Type:    "desktop",
		Enabled: true,
		Config:  make(map[string]interface{}),
		Filters: make([]*NotificationFilter, 0),
	}

	// Console notifications (for development)
	nm.config.Channels["console"] = &NotificationChannel{
		ID:      "console",
		Name:    "Console Notifications",
		Type:    "console",
		Enabled: true,
		Config:  make(map[string]interface{}),
		Filters: make([]*NotificationFilter, 0),
	}
}

func (nm *NotificationManager) passesFilters(notification *Notification, filters []*NotificationFilter) bool {
	for _, filter := range filters {
		if !nm.evaluateFilter(notification, filter) {
			return false
		}
	}
	return true
}

func (nm *NotificationManager) evaluateFilter(notification *Notification, filter *NotificationFilter) bool {
	// Simple filter evaluation - in a real implementation this would be more sophisticated
	var fieldValue interface{}

	switch filter.Field {
	case "type":
		fieldValue = string(notification.Type)
	case "priority":
		fieldValue = string(notification.Priority)
	case "user_id":
		fieldValue = notification.UserID
	case "channel_id":
		fieldValue = notification.ChannelID
	case "category":
		fieldValue = notification.Category
	default:
		return true // Unknown field, pass through
	}

	result := false
	switch filter.Operator {
	case "equals":
		result = fieldValue == filter.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if filterStr, ok := filter.Value.(string); ok {
				result = strings.Contains(str, filterStr)
			}
		}
	case "in":
		if values, ok := filter.Value.([]interface{}); ok {
			for _, value := range values {
				if fieldValue == value {
					result = true
					break
				}
			}
		}
	case "not_in":
		if values, ok := filter.Value.([]interface{}); ok {
			result = true
			for _, value := range values {
				if fieldValue == value {
					result = false
					break
				}
			}
		}
	default:
		result = true
	}

	if filter.Negate {
		result = !result
	}

	return result
}

func (nm *NotificationManager) isQuietHours() bool {
	if nm.config.QuietHours == nil || !nm.config.QuietHours.Enabled {
		return false
	}

	// Simple quiet hours check - in a real implementation this would handle timezones properly
	now := time.Now()
	currentTime := now.Format("15:04")

	return currentTime >= nm.config.QuietHours.StartTime && currentTime <= nm.config.QuietHours.EndTime
}

func (nm *NotificationManager) scheduleNotification(notification *Notification) error {
	// Simple scheduling - in a real implementation this would use a proper scheduler
	nm.logger.Debug("Scheduling notification for later delivery: %s", notification.ID)
	return nil
}

func (nm *NotificationManager) matchesFilters(notification *Notification, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "type":
			if string(notification.Type) != value {
				return false
			}
		case "status":
			if string(notification.Status) != value {
				return false
			}
		case "priority":
			if string(notification.Priority) != value {
				return false
			}
		case "user_id":
			if notification.UserID != value {
				return false
			}
		}
	}
	return true
}

func (nm *NotificationManager) updateBadgeCount() {
	count := 0
	for _, notification := range nm.notifications {
		if notification.Status == StatusPending || notification.Status == StatusDelivered {
			count++
		}
	}
	nm.badgeCount = count
}

func (nm *NotificationManager) getChannelsForNotification(notification *Notification) []string {
	// Simple channel selection - in a real implementation this would be more sophisticated
	return []string{nm.config.DefaultChannel}
}

func (nm *NotificationManager) isRateLimited(channelID string, rateLimit *RateLimit) bool {
	// Simple rate limiting check - in a real implementation this would track actual rates
	return false
}

func (nm *NotificationManager) handleDeliverySuccess(notification *Notification, channel *NotificationChannel) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification.Status = StatusDelivered
	now := time.Now()
	notification.DeliveredAt = &now
	notification.UpdatedAt = now

	// Update badge count
	if nm.config.BadgeCount {
		nm.updateBadgeCount()
	}

	nm.logger.Debug("Notification delivered successfully via %s: %s", channel.Type, notification.ID)
}

func (nm *NotificationManager) handleDeliveryFailure(notification *Notification, channel *NotificationChannel, err error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	notification.RetryCount++
	now := time.Now()
	notification.FailedAt = &now
	notification.UpdatedAt = now

	if notification.RetryCount >= notification.MaxRetries {
		notification.Status = StatusFailed
		nm.logger.Error("Notification delivery failed permanently: %s", notification.ID)
	} else {
		// Schedule retry
		nm.logger.Error("Notification delivery failed, will retry: %s (attempt %d/%d)",
			notification.ID, notification.RetryCount, notification.MaxRetries)
	}
}

func (nm *NotificationManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.performCleanup()
		case <-nm.ctx.Done():
			return
		}
	}
}

func (nm *NotificationManager) performCleanup() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	now := time.Now()
	cutoff := now.AddDate(0, 0, -nm.config.RetentionDays)
	var toDelete []string

	for id, notification := range nm.notifications {
		// Remove expired notifications
		if notification.ExpiresAt != nil && now.After(*notification.ExpiresAt) {
			toDelete = append(toDelete, id)
			continue
		}

		// Remove old notifications
		if notification.CreatedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
			continue
		}
	}

	// Delete notifications
	for _, id := range toDelete {
		delete(nm.notifications, id)
		if err := nm.storage.DeleteNotification(id); err != nil {
			nm.logger.Error("Failed to delete notification from storage: %v", err)
		}
	}

	if len(toDelete) > 0 {
		nm.logger.Info("Cleaned up %d expired/old notifications", len(toDelete))
	}

	// Enforce max notifications limit
	if len(nm.notifications) > nm.config.MaxNotifications {
		nm.enforceMaxNotifications()
	}
}

func (nm *NotificationManager) enforceMaxNotifications() {
	// Remove oldest notifications to stay within limit
	type notificationWithTime struct {
		id   string
		time time.Time
	}

	var notifications []notificationWithTime
	for id, notification := range nm.notifications {
		notifications = append(notifications, notificationWithTime{
			id:   id,
			time: notification.CreatedAt,
		})
	}

	// Sort by creation time (oldest first)
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].time.Before(notifications[j].time)
	})

	// Remove excess notifications
	excess := len(notifications) - nm.config.MaxNotifications
	for i := 0; i < excess; i++ {
		id := notifications[i].id
		delete(nm.notifications, id)
		if err := nm.storage.DeleteNotification(id); err != nil {
			nm.logger.Error("Failed to delete notification from storage: %v", err)
		}
	}

	nm.logger.Info("Removed %d notifications to enforce max limit", excess)
}

// Shutdown gracefully shuts down the notification manager
func (nm *NotificationManager) Shutdown() {
	nm.logger.Info("Shutting down notification manager...")
	nm.cancel()

	// Close storage
	if nm.storage != nil {
		nm.storage.Close()
	}

	nm.logger.Info("Notification manager shutdown complete")
}

// Helper functions
func generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

// NotificationStorage handles persistent storage of notifications
type NotificationStorage struct {
	baseDir string
	logger  *logging.Logger
	mu      sync.RWMutex
}

// NewNotificationStorage creates a new notification storage instance
func NewNotificationStorage(baseDir string) *NotificationStorage {
	storage := &NotificationStorage{
		baseDir: baseDir,
		logger:  logging.NewLogger(logging.INFO, nil, true),
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		storage.logger.Error("Failed to create notifications directory: %v", err)
	}

	return storage
}

// StoreNotification stores a notification to disk
func (ns *NotificationStorage) StoreNotification(notification *Notification) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	filePath := filepath.Join(ns.baseDir, notification.ID+".json")
	data, err := json.MarshalIndent(notification, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write notification file: %w", err)
	}

	return nil
}

// UpdateNotification updates a notification on disk
func (ns *NotificationStorage) UpdateNotification(notification *Notification) error {
	return ns.StoreNotification(notification) // Same as store for file-based storage
}

// DeleteNotification deletes a notification from disk
func (ns *NotificationStorage) DeleteNotification(notificationID string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	filePath := filepath.Join(ns.baseDir, notificationID+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete notification file: %w", err)
	}

	return nil
}

// LoadNotifications loads all notifications from disk
func (ns *NotificationStorage) LoadNotifications() (map[string]*Notification, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	notifications := make(map[string]*Notification)

	entries, err := os.ReadDir(ns.baseDir)
	if err != nil {
		return notifications, fmt.Errorf("failed to read notifications directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(ns.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			ns.logger.Error("Failed to read notification file %s: %v", filePath, err)
			continue
		}

		var notification Notification
		if err := json.Unmarshal(data, &notification); err != nil {
			ns.logger.Error("Failed to parse notification file %s: %v", filePath, err)
			continue
		}

		notifications[notification.ID] = &notification
	}

	return notifications, nil
}

// Close closes the notification storage
func (ns *NotificationStorage) Close() error {
	ns.logger.Info("Notification storage closed")
	return nil
}
