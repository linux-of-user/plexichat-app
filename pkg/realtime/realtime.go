package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"plexichat-client/pkg/client"
	"plexichat-client/pkg/logging"
)

// EventType represents different types of real-time events
type EventType string

const (
	EventTypeMessage        EventType = "message"
	EventTypeTyping         EventType = "typing"
	EventTypePresence       EventType = "presence"
	EventTypeUserJoined     EventType = "user_joined"
	EventTypeUserLeft       EventType = "user_left"
	EventTypeMessageEdited  EventType = "message_edited"
	EventTypeMessageDeleted EventType = "message_deleted"
	EventTypeError          EventType = "error"
	EventTypeHeartbeat      EventType = "heartbeat"
)

// Event represents a real-time event
type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	RoomID    string      `json:"room_id,omitempty"`
}

// TypingEvent represents a typing indicator event
type TypingEvent struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
	RoomID   string `json:"room_id,omitempty"`
}

// PresenceEvent represents a user presence event
type PresenceEvent struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Status   string `json:"status"` // online, away, busy, offline
	LastSeen time.Time `json:"last_seen"`
}

// MessageEvent represents a message event
type MessageEvent struct {
	Message *client.Message `json:"message"`
	Action  string          `json:"action"` // new, edited, deleted
}

// EventHandler defines the interface for handling events
type EventHandler interface {
	HandleEvent(event *Event)
}

// EventHandlerFunc is a function adapter for EventHandler
type EventHandlerFunc func(event *Event)

func (f EventHandlerFunc) HandleEvent(event *Event) {
	f(event)
}

// RealtimeManager manages WebSocket connections and real-time events
type RealtimeManager struct {
	client       *client.Client
	conn         *websocket.Conn
	handlers     map[EventType][]EventHandler
	mu           sync.RWMutex
	logger       *logging.Logger
	connected    bool
	reconnecting bool
	stopChan     chan struct{}
	heartbeat    *time.Ticker
	typingUsers  map[string]*TypingEvent
	userPresence map[string]*PresenceEvent
}

// NewRealtimeManager creates a new real-time manager
func NewRealtimeManager(apiClient *client.Client) *RealtimeManager {
	return &RealtimeManager{
		client:       apiClient,
		handlers:     make(map[EventType][]EventHandler),
		logger:       logging.NewLogger(logging.INFO, nil, true),
		stopChan:     make(chan struct{}),
		typingUsers:  make(map[string]*TypingEvent),
		userPresence: make(map[string]*PresenceEvent),
	}
}

// Connect establishes a WebSocket connection
func (rm *RealtimeManager) Connect(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.connected {
		return fmt.Errorf("already connected")
	}

	rm.logger.Info("Connecting to WebSocket...")

	conn, err := rm.client.ConnectWebSocket(ctx, "/ws")
	if err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	rm.conn = conn
	rm.connected = true
	rm.logger.Info("WebSocket connected successfully")

	// Start message handling goroutine
	go rm.handleMessages()

	// Start heartbeat
	rm.startHeartbeat()

	return nil
}

// Disconnect closes the WebSocket connection
func (rm *RealtimeManager) Disconnect() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.connected {
		return nil
	}

	rm.logger.Info("Disconnecting WebSocket...")

	// Stop heartbeat
	if rm.heartbeat != nil {
		rm.heartbeat.Stop()
	}

	// Signal stop
	close(rm.stopChan)

	// Close connection
	if rm.conn != nil {
		rm.conn.Close()
	}

	rm.connected = false
	rm.logger.Info("WebSocket disconnected")

	return nil
}

// IsConnected returns whether the WebSocket is connected
func (rm *RealtimeManager) IsConnected() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.connected
}

// AddHandler adds an event handler for a specific event type
func (rm *RealtimeManager) AddHandler(eventType EventType, handler EventHandler) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.handlers[eventType] = append(rm.handlers[eventType], handler)
	rm.logger.Debug("Added handler for event type: %s", eventType)
}

// RemoveHandler removes an event handler (not implemented for simplicity)
func (rm *RealtimeManager) RemoveHandler(eventType EventType, handler EventHandler) {
	// Implementation would require handler comparison
	rm.logger.Debug("RemoveHandler not implemented")
}

// SendEvent sends an event through the WebSocket
func (rm *RealtimeManager) SendEvent(event *Event) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.connected || rm.conn == nil {
		return fmt.Errorf("not connected")
	}

	event.Timestamp = time.Now()
	
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = rm.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		rm.logger.Error("Failed to send event: %v", err)
		return fmt.Errorf("failed to send event: %w", err)
	}

	rm.logger.Debug("Sent event: %s", event.Type)
	return nil
}

// SendTypingIndicator sends a typing indicator
func (rm *RealtimeManager) SendTypingIndicator(isTyping bool, roomID string) error {
	event := &Event{
		Type: EventTypeTyping,
		Data: &TypingEvent{
			IsTyping: isTyping,
			RoomID:   roomID,
		},
	}

	return rm.SendEvent(event)
}

// SendPresenceUpdate sends a presence update
func (rm *RealtimeManager) SendPresenceUpdate(status string) error {
	event := &Event{
		Type: EventTypePresence,
		Data: &PresenceEvent{
			Status:   status,
			LastSeen: time.Now(),
		},
	}

	return rm.SendEvent(event)
}

// GetTypingUsers returns currently typing users
func (rm *RealtimeManager) GetTypingUsers(roomID string) []*TypingEvent {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var typing []*TypingEvent
	for _, event := range rm.typingUsers {
		if event.IsTyping && (roomID == "" || event.RoomID == roomID) {
			typing = append(typing, event)
		}
	}

	return typing
}

// GetUserPresence returns presence information for a user
func (rm *RealtimeManager) GetUserPresence(userID string) *PresenceEvent {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.userPresence[userID]
}

// GetAllPresence returns presence information for all users
func (rm *RealtimeManager) GetAllPresence() map[string]*PresenceEvent {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy to avoid race conditions
	presence := make(map[string]*PresenceEvent)
	for k, v := range rm.userPresence {
		presence[k] = v
	}

	return presence
}

// handleMessages handles incoming WebSocket messages
func (rm *RealtimeManager) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			rm.logger.Error("Message handler panicked: %v", r)
		}
	}()

	for {
		select {
		case <-rm.stopChan:
			return
		default:
			if rm.conn == nil {
				return
			}

			_, message, err := rm.conn.ReadMessage()
			if err != nil {
				rm.logger.Error("Failed to read WebSocket message: %v", err)
				
				// Try to reconnect
				go rm.reconnect()
				return
			}

			rm.processMessage(message)
		}
	}
}

// processMessage processes an incoming message
func (rm *RealtimeManager) processMessage(data []byte) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		rm.logger.Error("Failed to unmarshal event: %v", err)
		return
	}

	rm.logger.Debug("Received event: %s", event.Type)

	// Update internal state based on event type
	rm.updateInternalState(&event)

	// Dispatch to handlers
	rm.dispatchEvent(&event)
}

// updateInternalState updates internal state based on events
func (rm *RealtimeManager) updateInternalState(event *Event) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	switch event.Type {
	case EventTypeTyping:
		if typingData, ok := event.Data.(map[string]interface{}); ok {
			userID := fmt.Sprintf("%v", typingData["user_id"])
			username := fmt.Sprintf("%v", typingData["username"])
			isTyping := typingData["is_typing"].(bool)
			roomID := fmt.Sprintf("%v", typingData["room_id"])

			if isTyping {
				rm.typingUsers[userID] = &TypingEvent{
					UserID:   userID,
					Username: username,
					IsTyping: true,
					RoomID:   roomID,
				}
			} else {
				delete(rm.typingUsers, userID)
			}
		}

	case EventTypePresence:
		if presenceData, ok := event.Data.(map[string]interface{}); ok {
			userID := fmt.Sprintf("%v", presenceData["user_id"])
			username := fmt.Sprintf("%v", presenceData["username"])
			status := fmt.Sprintf("%v", presenceData["status"])
			
			rm.userPresence[userID] = &PresenceEvent{
				UserID:   userID,
				Username: username,
				Status:   status,
				LastSeen: time.Now(),
			}
		}
	}
}

// dispatchEvent dispatches an event to registered handlers
func (rm *RealtimeManager) dispatchEvent(event *Event) {
	rm.mu.RLock()
	handlers := rm.handlers[event.Type]
	rm.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("Event handler panicked: %v", r)
				}
			}()
			h.HandleEvent(event)
		}(handler)
	}
}

// startHeartbeat starts the heartbeat mechanism
func (rm *RealtimeManager) startHeartbeat() {
	rm.heartbeat = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-rm.heartbeat.C:
				event := &Event{
					Type: EventTypeHeartbeat,
					Data: map[string]interface{}{"ping": time.Now().Unix()},
				}
				
				if err := rm.SendEvent(event); err != nil {
					rm.logger.Error("Failed to send heartbeat: %v", err)
				}
				
			case <-rm.stopChan:
				return
			}
		}
	}()
}

// reconnect attempts to reconnect the WebSocket
func (rm *RealtimeManager) reconnect() {
	rm.mu.Lock()
	if rm.reconnecting {
		rm.mu.Unlock()
		return
	}
	rm.reconnecting = true
	rm.connected = false
	rm.mu.Unlock()

	rm.logger.Info("Attempting to reconnect WebSocket...")

	// Exponential backoff
	backoff := time.Second
	maxBackoff := 30 * time.Second
	
	for {
		select {
		case <-rm.stopChan:
			return
		default:
			time.Sleep(backoff)
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := rm.Connect(ctx)
			cancel()
			
			if err == nil {
				rm.mu.Lock()
				rm.reconnecting = false
				rm.mu.Unlock()
				rm.logger.Info("WebSocket reconnected successfully")
				return
			}
			
			rm.logger.Error("Reconnection failed: %v", err)
			
			// Increase backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}
