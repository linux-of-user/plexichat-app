package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"plexichat-client/pkg/database"
	"plexichat-client/pkg/logging"
	"plexichat-client/pkg/messaging"
	"plexichat-client/pkg/websocket"
)

// RealtimeManager manages real-time communication
type RealtimeManager struct {
	wsClient    *websocket.Client
	db          *database.Database
	processor   *messaging.MessageProcessor
	logger      *logging.Logger
	subscribers map[string][]EventSubscriber
	eventQueue  chan *Event
	mu          sync.RWMutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// EventType represents different types of real-time events
type EventType string

const (
	EventTypeMessage         EventType = "message"
	EventTypeMessageEdit     EventType = "message_edit"
	EventTypeMessageDelete   EventType = "message_delete"
	EventTypeUserJoin        EventType = "user_join"
	EventTypeUserLeave       EventType = "user_leave"
	EventTypeUserTyping      EventType = "user_typing"
	EventTypeUserStatus      EventType = "user_status"
	EventTypeChannelCreate   EventType = "channel_create"
	EventTypeChannelUpdate   EventType = "channel_update"
	EventTypeChannelDelete   EventType = "channel_delete"
	EventTypeReaction        EventType = "reaction"
	EventTypePresence        EventType = "presence"
	EventTypeNotification    EventType = "notification"
	EventTypeFileUpload      EventType = "file_upload"
	EventTypeSystemMessage   EventType = "system_message"
)

// Event represents a real-time event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	ChannelID string                 `json:"channel_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventSubscriber defines the interface for event subscribers
type EventSubscriber interface {
	OnEvent(event *Event) error
	GetEventTypes() []EventType
	GetID() string
}

// TypingIndicator represents a typing indicator
type TypingIndicator struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	ChannelID string    `json:"channel_id"`
	StartTime time.Time `json:"start_time"`
	Active    bool      `json:"active"`
}

// UserPresence represents user presence information
type UserPresence struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Status     string    `json:"status"`
	LastSeen   time.Time `json:"last_seen"`
	IsOnline   bool      `json:"is_online"`
	CustomText string    `json:"custom_text,omitempty"`
}

// NewRealtimeManager creates a new real-time manager
func NewRealtimeManager(wsClient *websocket.Client, db *database.Database) *RealtimeManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RealtimeManager{
		wsClient:    wsClient,
		db:          db,
		processor:   messaging.NewMessageProcessor(db),
		logger:      logging.NewLogger(logging.INFO, nil, true),
		subscribers: make(map[string][]EventSubscriber),
		eventQueue:  make(chan *Event, 1000),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the real-time manager
func (rm *RealtimeManager) Start() error {
	rm.mu.Lock()
	if rm.running {
		rm.mu.Unlock()
		return fmt.Errorf("real-time manager already running")
	}
	rm.running = true
	rm.mu.Unlock()

	// Start event processing goroutine
	go rm.processEvents()

	// Start WebSocket message handling
	go rm.handleWebSocketMessages()

	// Start periodic tasks
	go rm.periodicTasks()

	rm.logger.Info("Real-time manager started")
	return nil
}

// Stop stops the real-time manager
func (rm *RealtimeManager) Stop() error {
	rm.mu.Lock()
	if !rm.running {
		rm.mu.Unlock()
		return fmt.Errorf("real-time manager not running")
	}
	rm.running = false
	rm.mu.Unlock()

	rm.cancel()
	close(rm.eventQueue)

	rm.logger.Info("Real-time manager stopped")
	return nil
}

// Subscribe subscribes to real-time events
func (rm *RealtimeManager) Subscribe(subscriber EventSubscriber) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	subscriberID := subscriber.GetID()
	eventTypes := subscriber.GetEventTypes()

	for _, eventType := range eventTypes {
		if rm.subscribers[string(eventType)] == nil {
			rm.subscribers[string(eventType)] = make([]EventSubscriber, 0)
		}
		rm.subscribers[string(eventType)] = append(rm.subscribers[string(eventType)], subscriber)
	}

	rm.logger.Info("Subscriber %s registered for events: %v", subscriberID, eventTypes)
}

// Unsubscribe unsubscribes from real-time events
func (rm *RealtimeManager) Unsubscribe(subscriberID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for eventType, subscribers := range rm.subscribers {
		newSubscribers := make([]EventSubscriber, 0)
		for _, subscriber := range subscribers {
			if subscriber.GetID() != subscriberID {
				newSubscribers = append(newSubscribers, subscriber)
			}
		}
		rm.subscribers[eventType] = newSubscribers
	}

	rm.logger.Info("Subscriber %s unsubscribed", subscriberID)
}

// PublishEvent publishes an event to subscribers
func (rm *RealtimeManager) PublishEvent(event *Event) {
	select {
	case rm.eventQueue <- event:
		// Event queued successfully
	default:
		rm.logger.Error("Event queue full, dropping event: %s", event.ID)
	}
}

// SendMessage sends a message through WebSocket
func (rm *RealtimeManager) SendMessage(channelID, content string) error {
	message := &websocket.Message{
		Type:      websocket.MessageTypeChat,
		ChannelID: channelID,
		Content:   content,
		Timestamp: time.Now(),
	}

	return rm.wsClient.SendMessage(message)
}

// SendTypingIndicator sends a typing indicator
func (rm *RealtimeManager) SendTypingIndicator(channelID string, typing bool) error {
	message := &websocket.Message{
		Type:      websocket.MessageTypeTyping,
		ChannelID: channelID,
		Content:   fmt.Sprintf(`{"typing": %t}`, typing),
		Timestamp: time.Now(),
	}

	return rm.wsClient.SendMessage(message)
}

// UpdatePresence updates user presence
func (rm *RealtimeManager) UpdatePresence(status, customText string) error {
	presence := UserPresence{
		Status:     status,
		LastSeen:   time.Now(),
		IsOnline:   status == "online",
		CustomText: customText,
	}

	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to marshal presence: %w", err)
	}

	message := &websocket.Message{
		Type:      websocket.MessageTypePresence,
		Content:   string(data),
		Timestamp: time.Now(),
	}

	return rm.wsClient.SendMessage(message)
}

// processEvents processes events from the queue
func (rm *RealtimeManager) processEvents() {
	for {
		select {
		case <-rm.ctx.Done():
			return
		case event, ok := <-rm.eventQueue:
			if !ok {
				return
			}

			rm.mu.RLock()
			subscribers := rm.subscribers[string(event.Type)]
			rm.mu.RUnlock()

			for _, subscriber := range subscribers {
				go func(sub EventSubscriber, evt *Event) {
					if err := sub.OnEvent(evt); err != nil {
						rm.logger.Error("Subscriber %s error handling event %s: %v", 
							sub.GetID(), evt.ID, err)
					}
				}(subscriber, event)
			}
		}
	}
}

// handleWebSocketMessages handles incoming WebSocket messages
func (rm *RealtimeManager) handleWebSocketMessages() {
	for {
		select {
		case <-rm.ctx.Done():
			return
		default:
			// This would be implemented with actual WebSocket message receiving
			// For now, we'll simulate with a ticker
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// periodicTasks runs periodic maintenance tasks
func (rm *RealtimeManager) periodicTasks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.cleanupExpiredTypingIndicators()
			rm.updateUserPresence()
		}
	}
}

// cleanupExpiredTypingIndicators removes expired typing indicators
func (rm *RealtimeManager) cleanupExpiredTypingIndicators() {
	// Implementation would clean up typing indicators older than 5 seconds
	rm.logger.Debug("Cleaning up expired typing indicators")
}

// updateUserPresence updates user presence information
func (rm *RealtimeManager) updateUserPresence() {
	// Implementation would update user presence in database
	rm.logger.Debug("Updating user presence")
}

// GetActiveUsers returns currently active users
func (rm *RealtimeManager) GetActiveUsers(ctx context.Context) ([]*UserPresence, error) {
	users, err := rm.db.GetOnlineUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get online users: %w", err)
	}

	presence := make([]*UserPresence, len(users))
	for i, user := range users {
		presence[i] = &UserPresence{
			UserID:   user.ID,
			Username: user.Username,
			Status:   user.Status,
			LastSeen: user.LastSeen,
			IsOnline: user.Status == "online",
		}
	}

	return presence, nil
}

// GetChannelActivity returns recent activity for a channel
func (rm *RealtimeManager) GetChannelActivity(ctx context.Context, channelID string, limit int) ([]*messaging.ProcessedMessage, error) {
	messages, err := rm.db.GetMessages(ctx, channelID, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel messages: %w", err)
	}

	processed := make([]*messaging.ProcessedMessage, len(messages))
	for i, msg := range messages {
		processedMsg, err := rm.processor.ProcessMessage(ctx, msg)
		if err != nil {
			rm.logger.Error("Failed to process message %d: %v", msg.ID, err)
			continue
		}
		processed[i] = processedMsg
	}

	return processed, nil
}
