// Package realtime provides real-time communication capabilities
package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"plexichat-client/internal/interfaces"
	"plexichat-client/pkg/logging"

	"github.com/gorilla/websocket"
)

// WebSocketClient provides advanced WebSocket client functionality
type WebSocketClient struct {
	mu                sync.RWMutex
	conn              *websocket.Conn
	url               string
	headers           http.Header
	config            WebSocketConfig
	logger            interfaces.Logger
	eventBus          interfaces.EventBus
	messageHandlers   map[string]MessageHandler
	connectionState   ConnectionState
	reconnectAttempts int32
	lastPing          time.Time
	lastPong          time.Time
	metrics           *WebSocketMetrics
	messageQueue      *MessageQueue
	rateLimiter       interfaces.RateLimiter
	compressor        MessageCompressor
	encryptor         MessageEncryptor
	authenticator     WebSocketAuthenticator
	middleware        []WebSocketMiddleware
	hooks             map[string][]WebSocketHook
	stopCh            chan struct{}
	pingTicker        *time.Ticker
	reconnectTimer    *time.Timer
	started           bool
}

// WebSocketConfig contains WebSocket client configuration
type WebSocketConfig struct {
	URL                   string            `json:"url"`
	Headers               map[string]string `json:"headers"`
	Subprotocols          []string          `json:"subprotocols"`
	PingInterval          time.Duration     `json:"ping_interval"`
	PongTimeout           time.Duration     `json:"pong_timeout"`
	ReconnectEnabled      bool              `json:"reconnect_enabled"`
	ReconnectInterval     time.Duration     `json:"reconnect_interval"`
	MaxReconnectAttempts  int               `json:"max_reconnect_attempts"`
	ReconnectBackoff      float64           `json:"reconnect_backoff"`
	MaxReconnectInterval  time.Duration     `json:"max_reconnect_interval"`
	HandshakeTimeout      time.Duration     `json:"handshake_timeout"`
	ReadBufferSize        int               `json:"read_buffer_size"`
	WriteBufferSize       int               `json:"write_buffer_size"`
	MessageQueueSize      int               `json:"message_queue_size"`
	CompressionEnabled    bool              `json:"compression_enabled"`
	EncryptionEnabled     bool              `json:"encryption_enabled"`
	AuthenticationEnabled bool              `json:"authentication_enabled"`
	RateLimitEnabled      bool              `json:"rate_limit_enabled"`
	MetricsEnabled        bool              `json:"metrics_enabled"`
	DebugEnabled          bool              `json:"debug_enabled"`
}

// ConnectionState represents WebSocket connection states
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateClosing
	StateClosed
	StateError
)

// MessageHandler handles incoming WebSocket messages
type MessageHandler interface {
	Handle(ctx context.Context, message *WebSocketMessage) error
	GetMessageType() string
	GetPriority() int
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id,omitempty"`
	Data       interface{}            `json:"data"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Compressed bool                   `json:"compressed,omitempty"`
	Encrypted  bool                   `json:"encrypted,omitempty"`
}

// MessageQueue manages outgoing message queue
type MessageQueue struct {
	mu       sync.RWMutex
	messages []*QueuedMessage
	maxSize  int
	priority bool
}

// QueuedMessage represents a queued message
type QueuedMessage struct {
	Message    *WebSocketMessage
	Priority   int
	Retries    int
	MaxRetries int
	Timestamp  time.Time
	Callback   func(error)
}

// MessageCompressor compresses/decompresses messages
type MessageCompressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	IsEnabled() bool
}

// MessageEncryptor encrypts/decrypts messages
type MessageEncryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	IsEnabled() bool
}

// WebSocketAuthenticator handles WebSocket authentication
type WebSocketAuthenticator interface {
	Authenticate(ctx context.Context, conn *websocket.Conn) error
	GetAuthHeaders() http.Header
	RefreshAuth(ctx context.Context) error
	IsAuthenticated() bool
}

// WebSocketMiddleware provides middleware for WebSocket operations
type WebSocketMiddleware interface {
	ProcessIncoming(ctx context.Context, message *WebSocketMessage, next func(*WebSocketMessage) error) error
	ProcessOutgoing(ctx context.Context, message *WebSocketMessage, next func(*WebSocketMessage) error) error
	GetName() string
}

// WebSocketHook provides hooks for WebSocket events
type WebSocketHook interface {
	OnConnect(ctx context.Context, conn *websocket.Conn) error
	OnDisconnect(ctx context.Context, err error) error
	OnMessage(ctx context.Context, message *WebSocketMessage) error
	OnError(ctx context.Context, err error) error
	GetName() string
}

// WebSocketMetrics tracks WebSocket metrics
type WebSocketMetrics struct {
	ConnectionsTotal   int64     `json:"connections_total"`
	ConnectionsActive  int64     `json:"connections_active"`
	MessagesReceived   int64     `json:"messages_received"`
	MessagesSent       int64     `json:"messages_sent"`
	MessagesQueued     int64     `json:"messages_queued"`
	MessagesFailed     int64     `json:"messages_failed"`
	ReconnectAttempts  int64     `json:"reconnect_attempts"`
	BytesReceived      int64     `json:"bytes_received"`
	BytesSent          int64     `json:"bytes_sent"`
	AverageLatency     int64     `json:"average_latency_ms"`
	ConnectionDuration int64     `json:"connection_duration_ms"`
	LastConnected      time.Time `json:"last_connected"`
	LastDisconnected   time.Time `json:"last_disconnected"`
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(config WebSocketConfig, eventBus interfaces.EventBus) *WebSocketClient {
	client := &WebSocketClient{
		url:             config.URL,
		headers:         make(http.Header),
		config:          config,
		logger:          logging.GetLogger("websocket"),
		eventBus:        eventBus,
		messageHandlers: make(map[string]MessageHandler),
		connectionState: StateDisconnected,
		metrics:         &WebSocketMetrics{},
		messageQueue:    NewMessageQueue(config.MessageQueueSize),
		middleware:      make([]WebSocketMiddleware, 0),
		hooks:           make(map[string][]WebSocketHook),
		stopCh:          make(chan struct{}),
	}

	// Set default values
	if client.config.PingInterval == 0 {
		client.config.PingInterval = 30 * time.Second
	}
	if client.config.PongTimeout == 0 {
		client.config.PongTimeout = 10 * time.Second
	}
	if client.config.ReconnectInterval == 0 {
		client.config.ReconnectInterval = 5 * time.Second
	}
	if client.config.MaxReconnectAttempts == 0 {
		client.config.MaxReconnectAttempts = 10
	}
	if client.config.ReconnectBackoff == 0 {
		client.config.ReconnectBackoff = 1.5
	}
	if client.config.MaxReconnectInterval == 0 {
		client.config.MaxReconnectInterval = 5 * time.Minute
	}
	if client.config.HandshakeTimeout == 0 {
		client.config.HandshakeTimeout = 10 * time.Second
	}

	// Set headers
	for key, value := range config.Headers {
		client.headers.Set(key, value)
	}

	return client
}

// Connect establishes a WebSocket connection
func (ws *WebSocketClient) Connect(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.connectionState == StateConnected || ws.connectionState == StateConnecting {
		return fmt.Errorf("already connected or connecting")
	}

	ws.logger.Info("Connecting to WebSocket", "url", ws.url)
	ws.connectionState = StateConnecting

	// Parse URL
	u, err := url.Parse(ws.url)
	if err != nil {
		ws.connectionState = StateError
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	// Create dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: ws.config.HandshakeTimeout,
		ReadBufferSize:   ws.config.ReadBufferSize,
		WriteBufferSize:  ws.config.WriteBufferSize,
		Subprotocols:     ws.config.Subprotocols,
	}

	// Add authentication headers if enabled
	headers := ws.headers
	if ws.config.AuthenticationEnabled && ws.authenticator != nil {
		authHeaders := ws.authenticator.GetAuthHeaders()
		for key, values := range authHeaders {
			for _, value := range values {
				headers.Add(key, value)
			}
		}
	}

	// Establish connection
	conn, resp, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		ws.connectionState = StateError
		ws.logger.Error("Failed to connect to WebSocket", "error", err)
		return fmt.Errorf("failed to connect: %w", err)
	}

	if resp != nil {
		resp.Body.Close()
	}

	ws.conn = conn
	ws.connectionState = StateConnected
	ws.reconnectAttempts = 0
	ws.metrics.LastConnected = time.Now()
	atomic.AddInt64(&ws.metrics.ConnectionsTotal, 1)
	atomic.AddInt64(&ws.metrics.ConnectionsActive, 1)

	// Authenticate if enabled
	if ws.config.AuthenticationEnabled && ws.authenticator != nil {
		if err := ws.authenticator.Authenticate(ctx, conn); err != nil {
			ws.logger.Error("WebSocket authentication failed", "error", err)
			ws.disconnect()
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Execute connection hooks
	if err := ws.executeHooks("connect", func(hook WebSocketHook) error {
		return hook.OnConnect(ctx, conn)
	}); err != nil {
		ws.logger.Error("Connection hook failed", "error", err)
	}

	// Start background routines
	go ws.readLoop(ctx)
	go ws.writeLoop(ctx)
	go ws.pingLoop(ctx)

	// Publish connection event
	if ws.eventBus != nil {
		event := &WebSocketEvent{
			Type:      "websocket.connected",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"url": ws.url,
			},
		}
		ws.eventBus.Publish(ctx, event)
	}

	ws.logger.Info("WebSocket connected successfully", "url", ws.url)
	return nil
}

// Disconnect closes the WebSocket connection
func (ws *WebSocketClient) Disconnect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	return ws.disconnect()
}

// disconnect closes the connection (internal method)
func (ws *WebSocketClient) disconnect() error {
	if ws.connectionState == StateDisconnected || ws.connectionState == StateClosed {
		return nil
	}

	ws.logger.Info("Disconnecting WebSocket")
	ws.connectionState = StateClosing

	// Stop background routines
	if ws.stopCh != nil {
		close(ws.stopCh)
		ws.stopCh = make(chan struct{})
	}

	// Stop timers
	if ws.pingTicker != nil {
		ws.pingTicker.Stop()
	}
	if ws.reconnectTimer != nil {
		ws.reconnectTimer.Stop()
	}

	// Close connection
	if ws.conn != nil {
		ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		ws.conn.Close()
		ws.conn = nil
	}

	ws.connectionState = StateDisconnected
	ws.metrics.LastDisconnected = time.Now()
	atomic.AddInt64(&ws.metrics.ConnectionsActive, -1)

	// Execute disconnection hooks
	if err := ws.executeHooks("disconnect", func(hook WebSocketHook) error {
		return hook.OnDisconnect(context.Background(), nil)
	}); err != nil {
		ws.logger.Error("Disconnection hook failed", "error", err)
	}

	// Publish disconnection event
	if ws.eventBus != nil {
		event := &WebSocketEvent{
			Type:      "websocket.disconnected",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"url": ws.url,
			},
		}
		ws.eventBus.Publish(context.Background(), event)
	}

	ws.logger.Info("WebSocket disconnected")
	return nil
}

// SendMessage sends a message through the WebSocket
func (ws *WebSocketClient) SendMessage(ctx context.Context, message *WebSocketMessage) error {
	if ws.connectionState != StateConnected {
		return fmt.Errorf("not connected")
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Apply rate limiting if enabled
	if ws.config.RateLimitEnabled && ws.rateLimiter != nil {
		if !ws.rateLimiter.Allow(ctx) {
			return fmt.Errorf("rate limit exceeded")
		}
	}

	// Apply outgoing middleware
	err := ws.applyOutgoingMiddleware(ctx, message)
	if err != nil {
		return fmt.Errorf("middleware failed: %w", err)
	}

	// Queue message for sending
	queuedMessage := &QueuedMessage{
		Message:    message,
		Priority:   0,
		MaxRetries: 3,
		Timestamp:  time.Now(),
	}

	return ws.messageQueue.Enqueue(queuedMessage)
}

// RegisterMessageHandler registers a message handler
func (ws *WebSocketClient) RegisterMessageHandler(handler MessageHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.messageHandlers[handler.GetMessageType()] = handler
	ws.logger.Debug("Message handler registered", "type", handler.GetMessageType())
}

// UnregisterMessageHandler unregisters a message handler
func (ws *WebSocketClient) UnregisterMessageHandler(messageType string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	delete(ws.messageHandlers, messageType)
	ws.logger.Debug("Message handler unregistered", "type", messageType)
}

// AddMiddleware adds WebSocket middleware
func (ws *WebSocketClient) AddMiddleware(middleware WebSocketMiddleware) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.middleware = append(ws.middleware, middleware)
	ws.logger.Debug("WebSocket middleware added", "name", middleware.GetName())
}

// AddHook adds a WebSocket hook
func (ws *WebSocketClient) AddHook(eventType string, hook WebSocketHook) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.hooks[eventType] == nil {
		ws.hooks[eventType] = make([]WebSocketHook, 0)
	}

	ws.hooks[eventType] = append(ws.hooks[eventType], hook)
	ws.logger.Debug("WebSocket hook added", "event", eventType, "name", hook.GetName())
}

// GetConnectionState returns the current connection state
func (ws *WebSocketClient) GetConnectionState() ConnectionState {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connectionState
}

// GetMetrics returns WebSocket metrics
func (ws *WebSocketClient) GetMetrics() *WebSocketMetrics {
	return ws.metrics
}

// IsConnected returns whether the WebSocket is connected
func (ws *WebSocketClient) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connectionState == StateConnected
}

// readLoop handles incoming messages
func (ws *WebSocketClient) readLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			ws.logger.Error("Read loop panic", "panic", r)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		default:
		}

		if ws.conn == nil {
			return
		}

		// Set read deadline
		ws.conn.SetReadDeadline(time.Now().Add(ws.config.PongTimeout))

		messageType, data, err := ws.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.Error("WebSocket read error", "error", err)
			}
			ws.handleConnectionError(ctx, err)
			return
		}

		atomic.AddInt64(&ws.metrics.MessagesReceived, 1)
		atomic.AddInt64(&ws.metrics.BytesReceived, int64(len(data)))

		// Handle different message types
		switch messageType {
		case websocket.TextMessage, websocket.BinaryMessage:
			if err := ws.handleMessage(ctx, data); err != nil {
				ws.logger.Error("Failed to handle message", "error", err)
			}
		case websocket.PongMessage:
			ws.lastPong = time.Now()
		case websocket.CloseMessage:
			ws.logger.Info("Received close message")
			ws.disconnect()
			return
		}
	}
}

// writeLoop handles outgoing messages
func (ws *WebSocketClient) writeLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			ws.logger.Error("Write loop panic", "panic", r)
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		case <-ticker.C:
			if err := ws.processMessageQueue(ctx); err != nil {
				ws.logger.Error("Failed to process message queue", "error", err)
			}
		}
	}
}

// pingLoop sends periodic ping messages
func (ws *WebSocketClient) pingLoop(ctx context.Context) {
	ws.pingTicker = time.NewTicker(ws.config.PingInterval)
	defer ws.pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.stopCh:
			return
		case <-ws.pingTicker.C:
			if err := ws.sendPing(); err != nil {
				ws.logger.Error("Failed to send ping", "error", err)
				ws.handleConnectionError(ctx, err)
				return
			}
		}
	}
}

// handleMessage processes incoming messages
func (ws *WebSocketClient) handleMessage(ctx context.Context, data []byte) error {
	// Decompress if needed
	if ws.config.CompressionEnabled && ws.compressor != nil && ws.compressor.IsEnabled() {
		decompressed, err := ws.compressor.Decompress(data)
		if err != nil {
			return fmt.Errorf("decompression failed: %w", err)
		}
		data = decompressed
	}

	// Decrypt if needed
	if ws.config.EncryptionEnabled && ws.encryptor != nil && ws.encryptor.IsEnabled() {
		decrypted, err := ws.encryptor.Decrypt(data)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
		data = decrypted
	}

	// Parse message
	var message WebSocketMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Apply incoming middleware
	if err := ws.applyIncomingMiddleware(ctx, &message); err != nil {
		return fmt.Errorf("middleware failed: %w", err)
	}

	// Execute message hooks
	if err := ws.executeHooks("message", func(hook WebSocketHook) error {
		return hook.OnMessage(ctx, &message)
	}); err != nil {
		ws.logger.Error("Message hook failed", "error", err)
	}

	// Find and execute handler
	ws.mu.RLock()
	handler, exists := ws.messageHandlers[message.Type]
	ws.mu.RUnlock()

	if exists {
		if err := handler.Handle(ctx, &message); err != nil {
			ws.logger.Error("Message handler failed", "type", message.Type, "error", err)
			return err
		}
	} else {
		ws.logger.Debug("No handler for message type", "type", message.Type)
	}

	// Publish message event
	if ws.eventBus != nil {
		event := &WebSocketEvent{
			Type:      "websocket.message",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message": message,
			},
		}
		ws.eventBus.Publish(ctx, event)
	}

	return nil
}

// processMessageQueue processes the outgoing message queue
func (ws *WebSocketClient) processMessageQueue(ctx context.Context) error {
	if ws.conn == nil {
		return fmt.Errorf("not connected")
	}

	message := ws.messageQueue.Dequeue()
	if message == nil {
		return nil
	}

	// Serialize message
	data, err := json.Marshal(message.Message)
	if err != nil {
		atomic.AddInt64(&ws.metrics.MessagesFailed, 1)
		if message.Callback != nil {
			message.Callback(err)
		}
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Encrypt if needed
	if ws.config.EncryptionEnabled && ws.encryptor != nil && ws.encryptor.IsEnabled() {
		encrypted, err := ws.encryptor.Encrypt(data)
		if err != nil {
			atomic.AddInt64(&ws.metrics.MessagesFailed, 1)
			if message.Callback != nil {
				message.Callback(err)
			}
			return fmt.Errorf("encryption failed: %w", err)
		}
		data = encrypted
		message.Message.Encrypted = true
	}

	// Compress if needed
	if ws.config.CompressionEnabled && ws.compressor != nil && ws.compressor.IsEnabled() {
		compressed, err := ws.compressor.Compress(data)
		if err != nil {
			atomic.AddInt64(&ws.metrics.MessagesFailed, 1)
			if message.Callback != nil {
				message.Callback(err)
			}
			return fmt.Errorf("compression failed: %w", err)
		}
		data = compressed
		message.Message.Compressed = true
	}

	// Send message
	if err := ws.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		atomic.AddInt64(&ws.metrics.MessagesFailed, 1)

		// Retry if configured
		if message.Retries < message.MaxRetries {
			message.Retries++
			ws.messageQueue.Enqueue(message)
		} else if message.Callback != nil {
			message.Callback(err)
		}

		return fmt.Errorf("failed to send message: %w", err)
	}

	atomic.AddInt64(&ws.metrics.MessagesSent, 1)
	atomic.AddInt64(&ws.metrics.BytesSent, int64(len(data)))

	if message.Callback != nil {
		message.Callback(nil)
	}

	return nil
}

// sendPing sends a ping message
func (ws *WebSocketClient) sendPing() error {
	if ws.conn == nil {
		return fmt.Errorf("not connected")
	}

	ws.lastPing = time.Now()
	return ws.conn.WriteMessage(websocket.PingMessage, []byte{})
}

// handleConnectionError handles connection errors
func (ws *WebSocketClient) handleConnectionError(ctx context.Context, err error) {
	ws.logger.Error("WebSocket connection error", "error", err)

	// Execute error hooks
	if hookErr := ws.executeHooks("error", func(hook WebSocketHook) error {
		return hook.OnError(ctx, err)
	}); hookErr != nil {
		ws.logger.Error("Error hook failed", "error", hookErr)
	}

	// Publish error event
	if ws.eventBus != nil {
		event := &WebSocketEvent{
			Type:      "websocket.error",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		}
		ws.eventBus.Publish(ctx, event)
	}

	// Attempt reconnection if enabled
	if ws.config.ReconnectEnabled {
		go ws.attemptReconnect(ctx)
	}
}

// attemptReconnect attempts to reconnect to the WebSocket
func (ws *WebSocketClient) attemptReconnect(ctx context.Context) {
	ws.mu.Lock()
	if ws.connectionState == StateReconnecting {
		ws.mu.Unlock()
		return
	}
	ws.connectionState = StateReconnecting
	ws.mu.Unlock()

	attempts := atomic.AddInt32(&ws.reconnectAttempts, 1)
	if int(attempts) > ws.config.MaxReconnectAttempts {
		ws.logger.Error("Max reconnect attempts reached", "attempts", attempts)
		ws.mu.Lock()
		ws.connectionState = StateError
		ws.mu.Unlock()
		return
	}

	// Calculate backoff delay
	delay := time.Duration(float64(ws.config.ReconnectInterval) *
		float64(attempts) * ws.config.ReconnectBackoff)
	if delay > ws.config.MaxReconnectInterval {
		delay = ws.config.MaxReconnectInterval
	}

	ws.logger.Info("Attempting to reconnect", "attempt", attempts, "delay", delay)
	atomic.AddInt64(&ws.metrics.ReconnectAttempts, 1)

	ws.reconnectTimer = time.NewTimer(delay)
	defer ws.reconnectTimer.Stop()

	select {
	case <-ws.reconnectTimer.C:
		if err := ws.Connect(ctx); err != nil {
			ws.logger.Error("Reconnect failed", "error", err)
			go ws.attemptReconnect(ctx)
		} else {
			ws.logger.Info("Reconnected successfully")
		}
	case <-ctx.Done():
		return
	}
}

// Helper functions and types

// WebSocketEvent represents a WebSocket event
type WebSocketEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// GetID returns the event ID
func (e *WebSocketEvent) GetID() string {
	if e.ID == "" {
		e.ID = fmt.Sprintf("ws_%d", time.Now().UnixNano())
	}
	return e.ID
}

// GetType returns the event type
func (e *WebSocketEvent) GetType() string {
	return e.Type
}

// GetSource returns the event source
func (e *WebSocketEvent) GetSource() string {
	if e.Source == "" {
		e.Source = "websocket"
	}
	return e.Source
}

// GetTimestamp returns the event timestamp
func (e *WebSocketEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetData returns the event data
func (e *WebSocketEvent) GetData() interface{} {
	return e.Data
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(maxSize int) *MessageQueue {
	return &MessageQueue{
		messages: make([]*QueuedMessage, 0),
		maxSize:  maxSize,
		priority: true,
	}
}

// Enqueue adds a message to the queue
func (mq *MessageQueue) Enqueue(message *QueuedMessage) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) >= mq.maxSize {
		// Remove oldest message if queue is full
		mq.messages = mq.messages[1:]
	}

	mq.messages = append(mq.messages, message)

	// Sort by priority if enabled
	if mq.priority {
		mq.sortByPriority()
	}

	return nil
}

// Dequeue removes and returns the next message from the queue
func (mq *MessageQueue) Dequeue() *QueuedMessage {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) == 0 {
		return nil
	}

	message := mq.messages[0]
	mq.messages = mq.messages[1:]
	return message
}

// Size returns the current queue size
func (mq *MessageQueue) Size() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return len(mq.messages)
}

// sortByPriority sorts messages by priority
func (mq *MessageQueue) sortByPriority() {
	for i := 0; i < len(mq.messages); i++ {
		for j := i + 1; j < len(mq.messages); j++ {
			if mq.messages[i].Priority < mq.messages[j].Priority {
				mq.messages[i], mq.messages[j] = mq.messages[j], mq.messages[i]
			}
		}
	}
}

// executeHooks executes hooks for a specific event type
func (ws *WebSocketClient) executeHooks(eventType string, executor func(WebSocketHook) error) error {
	ws.mu.RLock()
	hooks := ws.hooks[eventType]
	ws.mu.RUnlock()

	for _, hook := range hooks {
		if err := executor(hook); err != nil {
			return err
		}
	}

	return nil
}

// applyIncomingMiddleware applies middleware to incoming messages
func (ws *WebSocketClient) applyIncomingMiddleware(ctx context.Context, message *WebSocketMessage) error {
	if len(ws.middleware) == 0 {
		return nil
	}

	// Create middleware chain
	var next func(*WebSocketMessage) error
	next = func(msg *WebSocketMessage) error {
		return nil // Final handler does nothing
	}

	// Apply middleware in reverse order
	for i := len(ws.middleware) - 1; i >= 0; i-- {
		middleware := ws.middleware[i]
		currentNext := next
		next = func(msg *WebSocketMessage) error {
			return middleware.ProcessIncoming(ctx, msg, currentNext)
		}
	}

	return next(message)
}

// applyOutgoingMiddleware applies middleware to outgoing messages
func (ws *WebSocketClient) applyOutgoingMiddleware(ctx context.Context, message *WebSocketMessage) error {
	if len(ws.middleware) == 0 {
		return nil
	}

	// Create middleware chain
	var next func(*WebSocketMessage) error
	next = func(msg *WebSocketMessage) error {
		return nil // Final handler does nothing
	}

	// Apply middleware in reverse order
	for i := len(ws.middleware) - 1; i >= 0; i-- {
		middleware := ws.middleware[i]
		currentNext := next
		next = func(msg *WebSocketMessage) error {
			return middleware.ProcessOutgoing(ctx, msg, currentNext)
		}
	}

	return next(message)
}
