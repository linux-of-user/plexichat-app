package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	MessageTypeChat         MessageType = "chat"
	MessageTypePresence     MessageType = "presence"
	MessageTypeNotification MessageType = "notification"
	MessageTypeTyping       MessageType = "typing"
	MessageTypeJoin         MessageType = "join"
	MessageTypeLeave        MessageType = "leave"
	MessageTypeError        MessageType = "error"
	MessageTypePing         MessageType = "ping"
	MessageTypePong         MessageType = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	ChannelID string      `json:"channel_id,omitempty"`
	MessageID string      `json:"message_id,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   string
	Username string
	Conn     *websocket.Conn
	Send     chan Message
	Hub      *Hub
	Channels map[string]bool // Channels the client is subscribed to
	mu       sync.RWMutex
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients    map[string]*Client
	channels   map[string]map[string]*Client // channelID -> clientID -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		channels:   make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket hub shutting down")
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.pingClients()
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	log.Printf("Client %s (%s) connected", client.ID, client.Username)

	// Send welcome message
	welcomeMsg := Message{
		Type:      MessageTypeNotification,
		Data:      map[string]string{"message": "Connected to PlexiChat"},
		Timestamp: time.Now(),
	}
	
	select {
	case client.Send <- welcomeMsg:
	default:
		close(client.Send)
		delete(h.clients, client.ID)
	}
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		// Remove from all channels
		for channelID := range client.Channels {
			if channelClients, exists := h.channels[channelID]; exists {
				delete(channelClients, client.ID)
				if len(channelClients) == 0 {
					delete(h.channels, channelID)
				}
			}
		}

		delete(h.clients, client.ID)
		close(client.Send)
		log.Printf("Client %s (%s) disconnected", client.ID, client.Username)

		// Broadcast leave message to channels
		for channelID := range client.Channels {
			leaveMsg := Message{
				Type:      MessageTypeLeave,
				Data:      map[string]string{"username": client.Username},
				Timestamp: time.Now(),
				UserID:    client.UserID,
				ChannelID: channelID,
			}
			h.broadcastToChannel(channelID, leaveMsg)
		}
	}
}

// broadcastMessage broadcasts a message to appropriate clients
func (h *Hub) broadcastMessage(message Message) {
	if message.ChannelID != "" {
		h.broadcastToChannel(message.ChannelID, message)
	} else {
		h.broadcastToAll(message)
	}
}

// broadcastToChannel broadcasts message to all clients in a channel
func (h *Hub) broadcastToChannel(channelID string, message Message) {
	h.mu.RLock()
	channelClients, exists := h.channels[channelID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	for _, client := range channelClients {
		select {
		case client.Send <- message:
		default:
			h.unregister <- client
		}
	}
}

// broadcastToAll broadcasts message to all connected clients
func (h *Hub) broadcastToAll(message Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.Send <- message:
		default:
			h.unregister <- client
		}
	}
}

// pingClients sends ping messages to all clients
func (h *Hub) pingClients() {
	pingMsg := Message{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.Send <- pingMsg:
		default:
			h.unregister <- client
		}
	}
}

// JoinChannel adds a client to a channel
func (h *Hub) JoinChannel(clientID, channelID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found")
	}

	// Add to channel
	if h.channels[channelID] == nil {
		h.channels[channelID] = make(map[string]*Client)
	}
	h.channels[channelID][clientID] = client

	// Update client's channel list
	client.mu.Lock()
	client.Channels[channelID] = true
	client.mu.Unlock()

	// Broadcast join message
	joinMsg := Message{
		Type:      MessageTypeJoin,
		Data:      map[string]string{"username": client.Username},
		Timestamp: time.Now(),
		UserID:    client.UserID,
		ChannelID: channelID,
	}
	h.broadcastToChannel(channelID, joinMsg)

	log.Printf("Client %s joined channel %s", clientID, channelID)
	return nil
}

// LeaveChannel removes a client from a channel
func (h *Hub) LeaveChannel(clientID, channelID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found")
	}

	// Remove from channel
	if channelClients, exists := h.channels[channelID]; exists {
		delete(channelClients, clientID)
		if len(channelClients) == 0 {
			delete(h.channels, channelID)
		}
	}

	// Update client's channel list
	client.mu.Lock()
	delete(client.Channels, channelID)
	client.mu.Unlock()

	// Broadcast leave message
	leaveMsg := Message{
		Type:      MessageTypeLeave,
		Data:      map[string]string{"username": client.Username},
		Timestamp: time.Now(),
		UserID:    client.UserID,
		ChannelID: channelID,
	}
	h.broadcastToChannel(channelID, leaveMsg)

	log.Printf("Client %s left channel %s", clientID, channelID)
	return nil
}

// SendToChannel sends a message to a specific channel
func (h *Hub) SendToChannel(channelID string, message Message) {
	message.ChannelID = channelID
	message.Timestamp = time.Now()
	h.broadcast <- message
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, message Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				h.unregister <- client
			}
		}
	}
}

// GetChannelUsers returns list of users in a channel
func (h *Hub) GetChannelUsers(channelID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []string
	if channelClients, exists := h.channels[channelID]; exists {
		for _, client := range channelClients {
			users = append(users, client.Username)
		}
	}
	return users
}

// GetOnlineUsers returns list of all online users
func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var users []string
	for _, client := range h.clients {
		users = append(users, client.Username)
	}
	return users
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"total_clients":    len(h.clients),
		"total_channels":   len(h.channels),
		"timestamp":        time.Now(),
	}
}

// Upgrader configures the WebSocket upgrader
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// HandleWebSocket handles WebSocket upgrade and client management
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID, username string) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := fmt.Sprintf("%s_%d", userID, time.Now().UnixNano())
	client := &Client{
		ID:       clientID,
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan Message, 256),
		Hub:      h,
		Channels: make(map[string]bool),
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message Message
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Set message metadata
		message.UserID = c.UserID
		message.Timestamp = time.Now()

		// Handle different message types
		switch message.Type {
		case MessageTypeChat:
			c.Hub.broadcast <- message
		case MessageTypePong:
			// Handle pong response
		default:
			c.Hub.broadcast <- message
		}
	}
}
