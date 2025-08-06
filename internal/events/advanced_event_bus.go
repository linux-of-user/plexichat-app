// Package events provides advanced event bus and real-time communication
package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"plexichat-client/internal/interfaces"
	"plexichat-client/pkg/logging"
)

// AdvancedEventBus implements a sophisticated event bus with real-time capabilities
type AdvancedEventBus struct {
	mu                sync.RWMutex
	subscribers       map[string][]*Subscription
	eventHistory      *EventHistory
	eventFilters      []EventFilter
	eventTransformers []EventTransformer
	eventValidators   []EventValidator
	middleware        []EventMiddleware
	metrics           *EventBusMetrics
	logger            interfaces.Logger
	config            EventBusConfig
	channels          map[string]*EventChannel
	patterns          map[string]*PatternMatcher
	deadLetterQueue   *DeadLetterQueue
	retryPolicy       RetryPolicy
	circuitBreaker    *EventCircuitBreaker
	rateLimiter       EventRateLimiter
	serializer        EventSerializer
	compressor        EventCompressor
	encryptor         EventEncryptor
	batcher           *EventBatcher
	router            *EventRouter
	bridge            *EventBridge
	stopCh            chan struct{}
	started           bool
}

// Subscription represents an event subscription
type Subscription struct {
	ID         string
	EventType  string
	Pattern    string
	Handler    interfaces.EventHandler
	Filter     EventFilter
	Priority   int
	Async      bool
	Timeout    time.Duration
	RetryCount int
	MaxRetries int
	LastError  error
	Created    time.Time
	LastUsed   time.Time
	UsageCount int64
	Active     bool
	Metadata   map[string]interface{}
}

// EventHistory stores event history for replay and debugging
type EventHistory struct {
	mu      sync.RWMutex
	events  []*HistoricalEvent
	maxSize int
	ttl     time.Duration
	indices map[string][]int
}

// HistoricalEvent represents a stored event
type HistoricalEvent struct {
	Event     interfaces.Event
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// EventFilter filters events based on criteria
type EventFilter interface {
	Filter(ctx context.Context, event interfaces.Event) bool
	GetName() string
}

// EventTransformer transforms events
type EventTransformer interface {
	Transform(ctx context.Context, event interfaces.Event) (interfaces.Event, error)
	GetName() string
}

// EventValidator validates events
type EventValidator interface {
	Validate(ctx context.Context, event interfaces.Event) error
	GetName() string
}

// EventMiddleware provides middleware for event processing
type EventMiddleware interface {
	Process(ctx context.Context, event interfaces.Event, next func(interfaces.Event) error) error
	GetName() string
}

// EventChannel represents a communication channel
type EventChannel struct {
	Name        string
	Type        ChannelType
	Capacity    int
	Buffer      chan interfaces.Event
	Subscribers []*Subscription
	Config      ChannelConfig
	Stats       ChannelStats
	mu          sync.RWMutex
}

// ChannelType represents different channel types
type ChannelType int

const (
	ChannelTypeMemory ChannelType = iota
	ChannelTypePersistent
	ChannelTypeDistributed
	ChannelTypeRealTime
)

// ChannelConfig contains channel configuration
type ChannelConfig struct {
	Persistent    bool
	Ordered       bool
	Durable       bool
	Compressed    bool
	Encrypted     bool
	MaxSize       int
	TTL           time.Duration
	RetentionDays int
}

// ChannelStats contains channel statistics
type ChannelStats struct {
	MessagesPublished int64
	MessagesConsumed  int64
	ActiveSubscribers int
	AverageLatency    time.Duration
	ErrorCount        int64
}

// PatternMatcher matches event patterns
type PatternMatcher struct {
	Pattern string
	Regex   string
	Matcher func(string) bool
}

// DeadLetterQueue handles failed events
type DeadLetterQueue struct {
	mu       sync.RWMutex
	events   []*FailedEvent
	maxSize  int
	handlers []DeadLetterHandler
}

// FailedEvent represents a failed event
type FailedEvent struct {
	Event       interfaces.Event
	Error       error
	Attempts    int
	FirstFailed time.Time
	LastFailed  time.Time
	Metadata    map[string]interface{}
}

// DeadLetterHandler handles dead letter events
type DeadLetterHandler interface {
	Handle(ctx context.Context, event *FailedEvent) error
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// EventCircuitBreaker protects against cascading failures
type EventCircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failures         int64
	successes        int64
	lastFailure      time.Time
	nextRetry        time.Time
	failureThreshold int64
	timeout          time.Duration
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// EventRateLimiter limits event processing rate
type EventRateLimiter interface {
	Allow(ctx context.Context, eventType string) bool
	Wait(ctx context.Context, eventType string) error
}

// EventSerializer serializes/deserializes events
type EventSerializer interface {
	Serialize(event interfaces.Event) ([]byte, error)
	Deserialize(data []byte) (interfaces.Event, error)
	GetFormat() string
}

// EventCompressor compresses/decompresses events
type EventCompressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	GetAlgorithm() string
}

// EventEncryptor encrypts/decrypts events
type EventEncryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	GetAlgorithm() string
}

// EventBatcher batches events for efficient processing
type EventBatcher struct {
	mu      sync.RWMutex
	batches map[string]*EventBatch
	maxSize int
	maxWait time.Duration
	flushCh chan string
	stopCh  chan struct{}
}

// EventBatch represents a batch of events
type EventBatch struct {
	Events    []interfaces.Event
	StartTime time.Time
	Size      int
}

// EventRouter routes events to appropriate handlers
type EventRouter struct {
	mu     sync.RWMutex
	routes map[string]*Route
	rules  []RoutingRule
}

// Route represents an event route
type Route struct {
	Pattern     string
	Destination string
	Transform   EventTransformer
	Filter      EventFilter
	Priority    int
}

// RoutingRule defines routing logic
type RoutingRule interface {
	Match(event interfaces.Event) bool
	GetDestination(event interfaces.Event) string
	GetPriority() int
}

// EventBridge connects multiple event buses
type EventBridge struct {
	mu      sync.RWMutex
	bridges map[string]*BridgeConnection
	config  BridgeConfig
}

// BridgeConnection represents a bridge connection
type BridgeConnection struct {
	Name     string
	Type     BridgeType
	Endpoint string
	Config   map[string]interface{}
	Active   bool
	Stats    BridgeStats
}

// BridgeType represents bridge types
type BridgeType int

const (
	BridgeTypeHTTP BridgeType = iota
	BridgeTypeWebSocket
	BridgeTypeGRPC
	BridgeTypeKafka
	BridgeTypeRedis
	BridgeTypeNATS
)

// BridgeConfig contains bridge configuration
type BridgeConfig struct {
	Enabled       bool
	AutoReconnect bool
	Timeout       time.Duration
	BufferSize    int
}

// BridgeStats contains bridge statistics
type BridgeStats struct {
	EventsSent     int64
	EventsReceived int64
	Errors         int64
	LastActivity   time.Time
}

// EventBusConfig contains event bus configuration
type EventBusConfig struct {
	MaxSubscribers        int
	MaxEventHistory       int
	EventTTL              time.Duration
	AsyncProcessing       bool
	BatchProcessing       bool
	CompressionEnabled    bool
	EncryptionEnabled     bool
	MetricsEnabled        bool
	DeadLetterEnabled     bool
	CircuitBreakerEnabled bool
	RateLimitingEnabled   bool
	BridgingEnabled       bool
	HistoryEnabled        bool
}

// EventBusMetricsData contains event bus metrics
type EventBusMetricsData struct {
	EventsPublished   int64         `json:"events_published"`
	EventsProcessed   int64         `json:"events_processed"`
	EventsFailed      int64         `json:"events_failed"`
	ActiveSubscribers int           `json:"active_subscribers"`
	AverageLatency    time.Duration `json:"average_latency"`
	ThroughputPerSec  float64       `json:"throughput_per_sec"`
}

// NewAdvancedEventBus creates a new advanced event bus
func NewAdvancedEventBus(config EventBusConfig) *AdvancedEventBus {
	bus := &AdvancedEventBus{
		subscribers:       make(map[string][]*Subscription),
		eventHistory:      NewEventHistory(config.MaxEventHistory, config.EventTTL),
		eventFilters:      make([]EventFilter, 0),
		eventTransformers: make([]EventTransformer, 0),
		eventValidators:   make([]EventValidator, 0),
		middleware:        make([]EventMiddleware, 0),
		metrics:           NewEventBusMetrics(),
		logger:            logging.GetLogger("eventbus"),
		config:            config,
		channels:          make(map[string]*EventChannel),
		patterns:          make(map[string]*PatternMatcher),
		deadLetterQueue:   NewDeadLetterQueue(1000),
		retryPolicy: RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		},
		circuitBreaker: NewEventCircuitBreaker(10, 30*time.Second),
		batcher:        NewEventBatcher(100, 1*time.Second),
		router:         NewEventRouter(),
		bridge:         NewEventBridge(BridgeConfig{Enabled: config.BridgingEnabled}),
		stopCh:         make(chan struct{}),
	}

	// Start background processors
	if config.BatchProcessing {
		go bus.batcher.Start()
	}

	return bus
}

// Publish publishes an event to the bus
func (bus *AdvancedEventBus) Publish(ctx context.Context, event interfaces.Event) error {
	if !bus.started {
		return fmt.Errorf("event bus not started")
	}

	// Validate event
	for _, validator := range bus.eventValidators {
		if err := validator.Validate(ctx, event); err != nil {
			bus.logger.Error("Event validation failed", "validator", validator.GetName(), "error", err)
			return fmt.Errorf("event validation failed: %w", err)
		}
	}

	// Apply transformers
	transformedEvent := event
	for _, transformer := range bus.eventTransformers {
		var err error
		transformedEvent, err = transformer.Transform(ctx, transformedEvent)
		if err != nil {
			bus.logger.Error("Event transformation failed", "transformer", transformer.GetName(), "error", err)
			return fmt.Errorf("event transformation failed: %w", err)
		}
	}

	// Apply filters
	for _, filter := range bus.eventFilters {
		if !filter.Filter(ctx, transformedEvent) {
			bus.logger.Debug("Event filtered out", "filter", filter.GetName(), "event", transformedEvent.GetType())
			return nil
		}
	}

	// Check rate limiting
	if bus.rateLimiter != nil {
		if !bus.rateLimiter.Allow(ctx, transformedEvent.GetType()) {
			return fmt.Errorf("rate limit exceeded for event type: %s", transformedEvent.GetType())
		}
	}

	// Check circuit breaker
	if bus.circuitBreaker.IsOpen() {
		return fmt.Errorf("circuit breaker is open")
	}

	// Store in history if enabled
	if bus.config.HistoryEnabled {
		bus.eventHistory.Store(transformedEvent)
	}

	// Route event
	destinations := bus.router.Route(transformedEvent)
	if len(destinations) == 0 {
		destinations = []string{transformedEvent.GetType()}
	}

	// Publish to destinations
	for _, destination := range destinations {
		if err := bus.publishToDestination(ctx, destination, transformedEvent); err != nil {
			bus.logger.Error("Failed to publish to destination", "destination", destination, "error", err)
			bus.circuitBreaker.RecordFailure()

			// Add to dead letter queue
			if bus.config.DeadLetterEnabled {
				bus.deadLetterQueue.Add(&FailedEvent{
					Event:       transformedEvent,
					Error:       err,
					Attempts:    1,
					FirstFailed: time.Now(),
					LastFailed:  time.Now(),
				})
			}

			return err
		}
	}

	bus.circuitBreaker.RecordSuccess()
	atomic.AddInt64(&bus.metrics.EventsPublished, 1)

	bus.logger.Debug("Event published successfully", "type", transformedEvent.GetType(), "destinations", len(destinations))
	return nil
}

// Subscribe subscribes to events of a specific type
func (bus *AdvancedEventBus) Subscribe(eventType string, handler interfaces.EventHandler) (string, error) {
	return bus.SubscribeWithOptions(eventType, handler, SubscriptionOptions{})
}

// SubscriptionOptions contains subscription configuration
type SubscriptionOptions struct {
	Pattern    string
	Filter     EventFilter
	Priority   int
	Async      bool
	Timeout    time.Duration
	MaxRetries int
	Metadata   map[string]interface{}
}

// SubscribeWithOptions subscribes with advanced options
func (bus *AdvancedEventBus) SubscribeWithOptions(eventType string, handler interfaces.EventHandler, options SubscriptionOptions) (string, error) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	subscription := &Subscription{
		ID:         generateSubscriptionID(),
		EventType:  eventType,
		Pattern:    options.Pattern,
		Handler:    handler,
		Filter:     options.Filter,
		Priority:   options.Priority,
		Async:      options.Async,
		Timeout:    options.Timeout,
		MaxRetries: options.MaxRetries,
		Created:    time.Now(),
		Active:     true,
		Metadata:   options.Metadata,
	}

	if bus.subscribers[eventType] == nil {
		bus.subscribers[eventType] = make([]*Subscription, 0)
	}

	bus.subscribers[eventType] = append(bus.subscribers[eventType], subscription)

	// Sort by priority
	bus.sortSubscriptionsByPriority(eventType)

	bus.logger.Debug("Subscription created", "id", subscription.ID, "type", eventType, "priority", options.Priority)
	return subscription.ID, nil
}

// Unsubscribe removes a subscription
func (bus *AdvancedEventBus) Unsubscribe(subscriptionID string) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for eventType, subscriptions := range bus.subscribers {
		for i, subscription := range subscriptions {
			if subscription.ID == subscriptionID {
				// Remove subscription
				bus.subscribers[eventType] = append(subscriptions[:i], subscriptions[i+1:]...)
				bus.logger.Debug("Subscription removed", "id", subscriptionID, "type", eventType)
				return nil
			}
		}
	}

	return fmt.Errorf("subscription not found: %s", subscriptionID)
}

// publishToDestination publishes an event to a specific destination
func (bus *AdvancedEventBus) publishToDestination(ctx context.Context, destination string, event interfaces.Event) error {
	bus.mu.RLock()
	subscriptions := bus.subscribers[destination]
	bus.mu.RUnlock()

	if len(subscriptions) == 0 {
		return nil // No subscribers, not an error
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(subscriptions))

	for _, subscription := range subscriptions {
		if !subscription.Active {
			continue
		}

		// Apply subscription filter
		if subscription.Filter != nil && !subscription.Filter.Filter(ctx, event) {
			continue
		}

		if subscription.Async {
			wg.Add(1)
			go func(sub *Subscription) {
				defer wg.Done()
				if err := bus.processSubscription(ctx, sub, event); err != nil {
					errors <- err
				}
			}(subscription)
		} else {
			if err := bus.processSubscription(ctx, subscription, event); err != nil {
				errors <- err
			}
		}
	}

	if len(subscriptions) > 0 {
		go func() {
			wg.Wait()
			close(errors)
		}()
	} else {
		close(errors)
	}

	// Collect errors
	var lastError error
	for err := range errors {
		if err != nil {
			lastError = err
			bus.logger.Error("Subscription processing failed", "destination", destination, "error", err)
		}
	}

	return lastError
}

// processSubscription processes a single subscription
func (bus *AdvancedEventBus) processSubscription(ctx context.Context, subscription *Subscription, event interfaces.Event) error {
	startTime := time.Now()

	// Apply timeout if specified
	if subscription.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, subscription.Timeout)
		defer cancel()
	}

	// Apply middleware
	err := bus.applyMiddleware(ctx, event, func(e interfaces.Event) error {
		return subscription.Handler.Handle(ctx, e)
	})

	// Update subscription stats
	subscription.LastUsed = time.Now()
	atomic.AddInt64(&subscription.UsageCount, 1)

	if err != nil {
		subscription.LastError = err
		subscription.RetryCount++

		// Retry if configured
		if subscription.RetryCount <= subscription.MaxRetries {
			return bus.retrySubscription(ctx, subscription, event)
		}

		atomic.AddInt64(&bus.metrics.EventsFailed, 1)
		return err
	}

	// Record latency
	latency := time.Since(startTime)
	bus.metrics.RecordLatency(latency)

	atomic.AddInt64(&bus.metrics.EventsProcessed, 1)
	return nil
}

// retrySubscription retries a failed subscription
func (bus *AdvancedEventBus) retrySubscription(ctx context.Context, subscription *Subscription, event interfaces.Event) error {
	delay := bus.calculateRetryDelay(subscription.RetryCount)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return bus.processSubscription(ctx, subscription, event)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// calculateRetryDelay calculates retry delay with exponential backoff
func (bus *AdvancedEventBus) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(bus.retryPolicy.InitialDelay) *
		float64(attempt) * bus.retryPolicy.BackoffFactor)

	if delay > bus.retryPolicy.MaxDelay {
		delay = bus.retryPolicy.MaxDelay
	}

	// Add jitter if enabled
	if bus.retryPolicy.Jitter {
		jitterFactor := float64(2*time.Now().UnixNano()%2 - 1)
		jitter := time.Duration(float64(delay) * 0.1 * jitterFactor)
		delay += jitter
	}

	return delay
}

// applyMiddleware applies middleware to event processing
func (bus *AdvancedEventBus) applyMiddleware(ctx context.Context, event interfaces.Event, handler func(interfaces.Event) error) error {
	if len(bus.middleware) == 0 {
		return handler(event)
	}

	// Create middleware chain
	var next func(interfaces.Event) error
	next = handler

	// Apply middleware in reverse order
	for i := len(bus.middleware) - 1; i >= 0; i-- {
		middleware := bus.middleware[i]
		currentNext := next
		next = func(e interfaces.Event) error {
			return middleware.Process(ctx, e, currentNext)
		}
	}

	return next(event)
}

// sortSubscriptionsByPriority sorts subscriptions by priority
func (bus *AdvancedEventBus) sortSubscriptionsByPriority(eventType string) {
	subscriptions := bus.subscribers[eventType]
	for i := 0; i < len(subscriptions); i++ {
		for j := i + 1; j < len(subscriptions); j++ {
			if subscriptions[i].Priority < subscriptions[j].Priority {
				subscriptions[i], subscriptions[j] = subscriptions[j], subscriptions[i]
			}
		}
	}
}

// Start starts the event bus
func (bus *AdvancedEventBus) Start(ctx context.Context) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.started {
		return fmt.Errorf("event bus already started")
	}

	bus.logger.Info("Starting advanced event bus")

	// Start bridge if enabled
	if bus.config.BridgingEnabled {
		if err := bus.bridge.Start(ctx); err != nil {
			return fmt.Errorf("failed to start event bridge: %w", err)
		}
	}

	// Start dead letter queue processor
	if bus.config.DeadLetterEnabled {
		go bus.deadLetterQueue.Start(ctx)
	}

	bus.started = true
	bus.logger.Info("Advanced event bus started successfully")
	return nil
}

// Stop stops the event bus
func (bus *AdvancedEventBus) Stop(ctx context.Context) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if !bus.started {
		return nil
	}

	bus.logger.Info("Stopping advanced event bus")

	// Stop background processors
	close(bus.stopCh)

	// Stop bridge
	if bus.bridge != nil {
		bus.bridge.Stop(ctx)
	}

	// Stop batcher
	if bus.batcher != nil {
		bus.batcher.Stop()
	}

	bus.started = false
	bus.logger.Info("Advanced event bus stopped")
	return nil
}

// GetMetrics returns event bus metrics
func (bus *AdvancedEventBus) GetMetrics() EventBusMetricsData {
	return bus.metrics.GetSnapshot()
}

// AddFilter adds an event filter
func (bus *AdvancedEventBus) AddFilter(filter EventFilter) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.eventFilters = append(bus.eventFilters, filter)
	bus.logger.Debug("Event filter added", "name", filter.GetName())
}

// AddTransformer adds an event transformer
func (bus *AdvancedEventBus) AddTransformer(transformer EventTransformer) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.eventTransformers = append(bus.eventTransformers, transformer)
	bus.logger.Debug("Event transformer added", "name", transformer.GetName())
}

// AddValidator adds an event validator
func (bus *AdvancedEventBus) AddValidator(validator EventValidator) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.eventValidators = append(bus.eventValidators, validator)
	bus.logger.Debug("Event validator added", "name", validator.GetName())
}

// AddMiddleware adds event middleware
func (bus *AdvancedEventBus) AddMiddleware(middleware EventMiddleware) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.middleware = append(bus.middleware, middleware)
	bus.logger.Debug("Event middleware added", "name", middleware.GetName())
}

// Helper functions and constructors

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// EventBusMetrics tracks event bus metrics
type EventBusMetrics struct {
	EventsPublished int64
	EventsProcessed int64
	EventsFailed    int64
	TotalLatency    int64
	LatencyCount    int64
}

// NewEventBusMetrics creates new event bus metrics
func NewEventBusMetrics() *EventBusMetrics {
	return &EventBusMetrics{}
}

// RecordLatency records processing latency
func (m *EventBusMetrics) RecordLatency(latency time.Duration) {
	atomic.AddInt64(&m.TotalLatency, int64(latency))
	atomic.AddInt64(&m.LatencyCount, 1)
}

// GetSnapshot returns a metrics snapshot
func (m *EventBusMetrics) GetSnapshot() EventBusMetricsData {
	published := atomic.LoadInt64(&m.EventsPublished)
	processed := atomic.LoadInt64(&m.EventsProcessed)
	failed := atomic.LoadInt64(&m.EventsFailed)
	totalLatency := atomic.LoadInt64(&m.TotalLatency)
	latencyCount := atomic.LoadInt64(&m.LatencyCount)

	var avgLatency time.Duration
	if latencyCount > 0 {
		avgLatency = time.Duration(totalLatency / latencyCount)
	}

	var throughput float64
	if processed > 0 {
		throughput = float64(processed) / time.Since(time.Now().Add(-time.Minute)).Seconds()
	}

	return EventBusMetricsData{
		EventsPublished:   published,
		EventsProcessed:   processed,
		EventsFailed:      failed,
		ActiveSubscribers: 0, // TODO: Calculate active subscribers
		AverageLatency:    avgLatency,
		ThroughputPerSec:  throughput,
	}
}

// NewEventHistory creates a new event history
func NewEventHistory(maxSize int, ttl time.Duration) *EventHistory {
	return &EventHistory{
		events:  make([]*HistoricalEvent, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
		indices: make(map[string][]int),
	}
}

// Store stores an event in history
func (eh *EventHistory) Store(event interfaces.Event) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	historicalEvent := &HistoricalEvent{
		Event:     event,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Add to events slice
	if len(eh.events) >= eh.maxSize {
		// Remove oldest event
		oldest := eh.events[0]
		eh.events = eh.events[1:]

		// Update indices
		eventType := oldest.Event.GetType()
		if indices, exists := eh.indices[eventType]; exists {
			// Remove first index and shift others
			if len(indices) > 0 {
				eh.indices[eventType] = indices[1:]
				// Shift all indices down by 1
				for i := range eh.indices[eventType] {
					eh.indices[eventType][i]--
				}
			}
		}
	}

	eh.events = append(eh.events, historicalEvent)

	// Update index
	eventType := event.GetType()
	if eh.indices[eventType] == nil {
		eh.indices[eventType] = make([]int, 0)
	}
	eh.indices[eventType] = append(eh.indices[eventType], len(eh.events)-1)
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(maxSize int) *DeadLetterQueue {
	return &DeadLetterQueue{
		events:   make([]*FailedEvent, 0, maxSize),
		maxSize:  maxSize,
		handlers: make([]DeadLetterHandler, 0),
	}
}

// Add adds a failed event to the dead letter queue
func (dlq *DeadLetterQueue) Add(event *FailedEvent) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	if len(dlq.events) >= dlq.maxSize {
		// Remove oldest event
		dlq.events = dlq.events[1:]
	}

	dlq.events = append(dlq.events, event)
}

// Start starts the dead letter queue processor
func (dlq *DeadLetterQueue) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dlq.processEvents(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// processEvents processes events in the dead letter queue
func (dlq *DeadLetterQueue) processEvents(ctx context.Context) {
	dlq.mu.RLock()
	events := make([]*FailedEvent, len(dlq.events))
	copy(events, dlq.events)
	dlq.mu.RUnlock()

	for _, event := range events {
		for _, handler := range dlq.handlers {
			if err := handler.Handle(ctx, event); err == nil {
				// Successfully handled, remove from queue
				dlq.remove(event)
				break
			}
		}
	}
}

// remove removes an event from the dead letter queue
func (dlq *DeadLetterQueue) remove(event *FailedEvent) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	for i, e := range dlq.events {
		if e == event {
			dlq.events = append(dlq.events[:i], dlq.events[i+1:]...)
			break
		}
	}
}

// NewEventCircuitBreaker creates a new event circuit breaker
func NewEventCircuitBreaker(failureThreshold int64, timeout time.Duration) *EventCircuitBreaker {
	return &EventCircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		timeout:          timeout,
	}
}

// IsOpen returns whether the circuit breaker is open
func (cb *EventCircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == CircuitOpen {
		if time.Now().After(cb.nextRetry) {
			cb.state = CircuitHalfOpen
			return false
		}
		return true
	}

	return false
}

// RecordSuccess records a successful operation
func (cb *EventCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failures = 0
	}
}

// RecordFailure records a failed operation
func (cb *EventCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
		cb.nextRetry = time.Now().Add(cb.timeout)
	}
}

// NewEventBatcher creates a new event batcher
func NewEventBatcher(maxSize int, maxWait time.Duration) *EventBatcher {
	return &EventBatcher{
		batches: make(map[string]*EventBatch),
		maxSize: maxSize,
		maxWait: maxWait,
		flushCh: make(chan string, 100),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the event batcher
func (eb *EventBatcher) Start() {
	ticker := time.NewTicker(eb.maxWait)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eb.flushAll()
		case batchKey := <-eb.flushCh:
			eb.flushBatch(batchKey)
		case <-eb.stopCh:
			return
		}
	}
}

// Stop stops the event batcher
func (eb *EventBatcher) Stop() {
	close(eb.stopCh)
}

// flushAll flushes all batches
func (eb *EventBatcher) flushAll() {
	eb.mu.RLock()
	keys := make([]string, 0, len(eb.batches))
	for key := range eb.batches {
		keys = append(keys, key)
	}
	eb.mu.RUnlock()

	for _, key := range keys {
		eb.flushBatch(key)
	}
}

// flushBatch flushes a specific batch
func (eb *EventBatcher) flushBatch(batchKey string) {
	eb.mu.Lock()
	batch, exists := eb.batches[batchKey]
	if exists {
		delete(eb.batches, batchKey)
	}
	eb.mu.Unlock()

	if exists && len(batch.Events) > 0 {
		// TODO: Process batch
	}
}

// NewEventRouter creates a new event router
func NewEventRouter() *EventRouter {
	return &EventRouter{
		routes: make(map[string]*Route),
		rules:  make([]RoutingRule, 0),
	}
}

// Route routes an event to destinations
func (er *EventRouter) Route(event interfaces.Event) []string {
	er.mu.RLock()
	defer er.mu.RUnlock()

	destinations := make([]string, 0)

	// Apply routing rules
	for _, rule := range er.rules {
		if rule.Match(event) {
			destination := rule.GetDestination(event)
			destinations = append(destinations, destination)
		}
	}

	// If no rules matched, use default routing
	if len(destinations) == 0 {
		destinations = append(destinations, event.GetType())
	}

	return destinations
}

// NewEventBridge creates a new event bridge
func NewEventBridge(config BridgeConfig) *EventBridge {
	return &EventBridge{
		bridges: make(map[string]*BridgeConnection),
		config:  config,
	}
}

// Start starts the event bridge
func (eb *EventBridge) Start(ctx context.Context) error {
	// TODO: Implement bridge startup
	return nil
}

// Stop stops the event bridge
func (eb *EventBridge) Stop(ctx context.Context) error {
	// TODO: Implement bridge shutdown
	return nil
}
