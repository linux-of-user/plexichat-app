package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"plexichat-client/pkg/logging"
)

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[string][]EventHandler
	middleware  []EventMiddleware
	logger      *logging.Logger
	mu          sync.RWMutex
	running     bool
	eventQueue  chan *Event
	ctx         context.Context
	cancel      context.CancelFunc
}

// Event represents an event in the system
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Context   context.Context        `json:"-"`
}

// EventHandler defines the interface for event handlers
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
	GetEventTypes() []string
	GetPriority() int
}

// EventMiddleware defines the interface for event middleware
type EventMiddleware interface {
	Process(ctx context.Context, event *Event, next func(*Event) error) error
	GetPriority() int
}

// EventSubscription represents an event subscription
type EventSubscription struct {
	ID            string
	EventTypes    []string
	Handler       EventHandler
	CreatedAt     time.Time
	LastTriggered *time.Time
	TriggerCount  int64
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		middleware:  make([]EventMiddleware, 0),
		logger:      logging.NewLogger(logging.INFO, nil, true),
		eventQueue:  make(chan *Event, 10000),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the event bus
func (eb *EventBus) Start() error {
	eb.mu.Lock()
	if eb.running {
		eb.mu.Unlock()
		return fmt.Errorf("event bus already running")
	}
	eb.running = true
	eb.mu.Unlock()

	// Start event processing goroutine
	go eb.processEvents()

	eb.logger.Info("Event bus started")
	return nil
}

// Stop stops the event bus
func (eb *EventBus) Stop() error {
	eb.mu.Lock()
	if !eb.running {
		eb.mu.Unlock()
		return fmt.Errorf("event bus not running")
	}
	eb.running = false
	eb.mu.Unlock()

	eb.cancel()
	close(eb.eventQueue)

	eb.logger.Info("Event bus stopped")
	return nil
}

// Subscribe subscribes a handler to events
func (eb *EventBus) Subscribe(handler EventHandler) string {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscriptionID := eb.generateSubscriptionID()
	eventTypes := handler.GetEventTypes()

	for _, eventType := range eventTypes {
		if eb.subscribers[eventType] == nil {
			eb.subscribers[eventType] = make([]EventHandler, 0)
		}
		eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
	}

	// Sort handlers by priority
	for eventType := range eb.subscribers {
		eb.sortHandlersByPriority(eventType)
	}

	eb.logger.Info("Subscribed handler %s to events: %v", subscriptionID, eventTypes)
	return subscriptionID
}

// Unsubscribe removes a handler from events
func (eb *EventBus) Unsubscribe(subscriptionID string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eventTypes := handler.GetEventTypes()
	for _, eventType := range eventTypes {
		handlers := eb.subscribers[eventType]
		newHandlers := make([]EventHandler, 0)

		for _, h := range handlers {
			if h != handler {
				newHandlers = append(newHandlers, h)
			}
		}

		eb.subscribers[eventType] = newHandlers
	}

	eb.logger.Info("Unsubscribed handler %s", subscriptionID)
}

// Publish publishes an event
func (eb *EventBus) Publish(event *Event) error {
	if event.ID == "" {
		event.ID = eb.generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Context == nil {
		event.Context = context.Background()
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	select {
	case eb.eventQueue <- event:
		return nil
	default:
		eb.logger.Error("Event queue full, dropping event: %s", event.ID)
		return fmt.Errorf("event queue full")
	}
}

// PublishSync publishes an event synchronously
func (eb *EventBus) PublishSync(event *Event) error {
	if event.ID == "" {
		event.ID = eb.generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Context == nil {
		event.Context = context.Background()
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}

	return eb.handleEvent(event)
}

// AddMiddleware adds middleware to the event bus
func (eb *EventBus) AddMiddleware(middleware EventMiddleware) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.middleware = append(eb.middleware, middleware)

	// Sort middleware by priority
	for i := 0; i < len(eb.middleware)-1; i++ {
		for j := i + 1; j < len(eb.middleware); j++ {
			if eb.middleware[i].GetPriority() > eb.middleware[j].GetPriority() {
				eb.middleware[i], eb.middleware[j] = eb.middleware[j], eb.middleware[i]
			}
		}
	}

	eb.logger.Info("Added middleware with priority %d", middleware.GetPriority())
}

// processEvents processes events from the queue
func (eb *EventBus) processEvents() {
	for {
		select {
		case <-eb.ctx.Done():
			return
		case event, ok := <-eb.eventQueue:
			if !ok {
				return
			}

			if err := eb.handleEvent(event); err != nil {
				eb.logger.Error("Failed to handle event %s: %v", event.ID, err)
			}
		}
	}
}

// handleEvent handles a single event
func (eb *EventBus) handleEvent(event *Event) error {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	middleware := make([]EventMiddleware, len(eb.middleware))
	copy(middleware, eb.middleware)
	eb.mu.RUnlock()

	// Apply middleware
	var processFunc func(*Event) error
	processFunc = func(e *Event) error {
		// Execute handlers
		for _, handler := range handlers {
			if err := handler.Handle(e.Context, e); err != nil {
				eb.logger.Error("Handler error for event %s: %v", e.ID, err)
				// Continue with other handlers
			}
		}
		return nil
	}

	// Apply middleware in reverse order
	for i := len(middleware) - 1; i >= 0; i-- {
		mw := middleware[i]
		nextFunc := processFunc
		processFunc = func(e *Event) error {
			return mw.Process(e.Context, e, nextFunc)
		}
	}

	return processFunc(event)
}

// sortHandlersByPriority sorts handlers by priority
func (eb *EventBus) sortHandlersByPriority(eventType string) {
	handlers := eb.subscribers[eventType]
	for i := 0; i < len(handlers)-1; i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].GetPriority() > handlers[j].GetPriority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}
}

// generateEventID generates a unique event ID
func (eb *EventBus) generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// generateSubscriptionID generates a unique subscription ID
func (eb *EventBus) generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// GetSubscriptions returns all active subscriptions
func (eb *EventBus) GetSubscriptions() map[string][]string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscriptions := make(map[string][]string)
	for eventType, handlers := range eb.subscribers {
		handlerNames := make([]string, len(handlers))
		for i, handler := range handlers {
			handlerNames[i] = reflect.TypeOf(handler).String()
		}
		subscriptions[eventType] = handlerNames
	}

	return subscriptions
}

// GetStats returns event bus statistics
func (eb *EventBus) GetStats() map[string]interface{} {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["running"] = eb.running
	stats["queue_size"] = len(eb.eventQueue)
	stats["queue_capacity"] = cap(eb.eventQueue)
	stats["subscriber_count"] = len(eb.subscribers)
	stats["middleware_count"] = len(eb.middleware)

	// Count total handlers
	totalHandlers := 0
	for _, handlers := range eb.subscribers {
		totalHandlers += len(handlers)
	}
	stats["total_handlers"] = totalHandlers

	return stats
}

// Built-in event handlers and middleware

// LoggingMiddleware logs all events
type LoggingMiddleware struct {
	logger *logging.Logger
}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

func (lm *LoggingMiddleware) Process(ctx context.Context, event *Event, next func(*Event) error) error {
	start := time.Now()
	lm.logger.Debug("Processing event: %s (type: %s, source: %s)", event.ID, event.Type, event.Source)

	err := next(event)

	duration := time.Since(start)
	if err != nil {
		lm.logger.Error("Event processing failed: %s (duration: %v, error: %v)", event.ID, duration, err)
	} else {
		lm.logger.Debug("Event processed successfully: %s (duration: %v)", event.ID, duration)
	}

	return err
}

func (lm *LoggingMiddleware) GetPriority() int {
	return 1000 // Low priority (runs last)
}

// MetricsMiddleware collects event metrics
type MetricsMiddleware struct {
	eventCounts map[string]int64
	mu          sync.RWMutex
}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		eventCounts: make(map[string]int64),
	}
}

func (mm *MetricsMiddleware) Process(ctx context.Context, event *Event, next func(*Event) error) error {
	mm.mu.Lock()
	mm.eventCounts[event.Type]++
	mm.mu.Unlock()

	return next(event)
}

func (mm *MetricsMiddleware) GetPriority() int {
	return 100 // High priority (runs early)
}

func (mm *MetricsMiddleware) GetEventCounts() map[string]int64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	counts := make(map[string]int64)
	for k, v := range mm.eventCounts {
		counts[k] = v
	}
	return counts
}

// ValidationMiddleware validates events
type ValidationMiddleware struct{}

func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{}
}

func (vm *ValidationMiddleware) Process(ctx context.Context, event *Event, next func(*Event) error) error {
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.Source == "" {
		return fmt.Errorf("event source is required")
	}

	return next(event)
}

func (vm *ValidationMiddleware) GetPriority() int {
	return 1 // Highest priority (runs first)
}

// BaseEventHandler provides common functionality for event handlers
type BaseEventHandler struct {
	eventTypes []string
	priority   int
}

func (beh *BaseEventHandler) GetEventTypes() []string {
	return beh.eventTypes
}

func (beh *BaseEventHandler) GetPriority() int {
	return beh.priority
}

// SystemEventHandler handles system events
type SystemEventHandler struct {
	BaseEventHandler
	logger *logging.Logger
}

func NewSystemEventHandler() *SystemEventHandler {
	return &SystemEventHandler{
		BaseEventHandler: BaseEventHandler{
			eventTypes: []string{"system.startup", "system.shutdown", "system.error"},
			priority:   50,
		},
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

func (seh *SystemEventHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case "system.startup":
		seh.logger.Info("System startup event received")
	case "system.shutdown":
		seh.logger.Info("System shutdown event received")
	case "system.error":
		seh.logger.Error("System error event: %v", event.Data)
	}
	return nil
}

// UserEventHandler handles user-related events
type UserEventHandler struct {
	BaseEventHandler
	logger *logging.Logger
}

func NewUserEventHandler() *UserEventHandler {
	return &UserEventHandler{
		BaseEventHandler: BaseEventHandler{
			eventTypes: []string{"user.login", "user.logout", "user.register", "user.update"},
			priority:   50,
		},
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

func (ueh *UserEventHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case "user.login":
		ueh.logger.Info("User login event: %v", event.Data)
	case "user.logout":
		ueh.logger.Info("User logout event: %v", event.Data)
	case "user.register":
		ueh.logger.Info("User registration event: %v", event.Data)
	case "user.update":
		ueh.logger.Info("User update event: %v", event.Data)
	}
	return nil
}

// MessageEventHandler handles message-related events
type MessageEventHandler struct {
	BaseEventHandler
	logger *logging.Logger
}

func NewMessageEventHandler() *MessageEventHandler {
	return &MessageEventHandler{
		BaseEventHandler: BaseEventHandler{
			eventTypes: []string{"message.sent", "message.received", "message.edited", "message.deleted"},
			priority:   50,
		},
		logger: logging.NewLogger(logging.INFO, nil, true),
	}
}

func (meh *MessageEventHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case "message.sent":
		meh.logger.Debug("Message sent event: %v", event.Data)
	case "message.received":
		meh.logger.Debug("Message received event: %v", event.Data)
	case "message.edited":
		meh.logger.Debug("Message edited event: %v", event.Data)
	case "message.deleted":
		meh.logger.Debug("Message deleted event: %v", event.Data)
	}
	return nil
}
