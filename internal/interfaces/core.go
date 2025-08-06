// Package interfaces defines the core interfaces for the PlexiChat client
package interfaces

import (
	"context"
	"io"
	"time"
)

// Application represents the main application interface
type Application interface {
	// Start starts the application
	Start(ctx context.Context) error

	// Stop stops the application gracefully
	Stop(ctx context.Context) error

	// GetVersion returns the application version
	GetVersion() string

	// GetBuildInfo returns build information
	GetBuildInfo() BuildInfo

	// IsHealthy returns the health status
	IsHealthy() bool
}

// BuildInfo contains build information
type BuildInfo struct {
	Version   string    `json:"version"`
	GitCommit string    `json:"git_commit"`
	BuildTime time.Time `json:"build_time"`
	GoVersion string    `json:"go_version"`
	Platform  string    `json:"platform"`
}

// ConfigManager manages application configuration
type ConfigManager interface {
	// Load loads configuration from various sources
	Load(ctx context.Context) error

	// Get retrieves a configuration value
	Get(key string) interface{}

	// GetString retrieves a string configuration value
	GetString(key string) string

	// GetInt retrieves an integer configuration value
	GetInt(key string) int

	// GetBool retrieves a boolean configuration value
	GetBool(key string) bool

	// GetDuration retrieves a duration configuration value
	GetDuration(key string) time.Duration

	// Set sets a configuration value
	Set(key string, value interface{}) error

	// Watch watches for configuration changes
	Watch(key string, callback func(interface{})) error

	// Validate validates the current configuration
	Validate() error

	// Save saves configuration to persistent storage
	Save() error

	// GetProfile returns the current configuration profile
	GetProfile() string

	// SetProfile sets the configuration profile
	SetProfile(profile string) error

	// ListProfiles returns available configuration profiles
	ListProfiles() []string
}

// Logger defines the logging interface
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...interface{})

	// Info logs an info message
	Info(msg string, fields ...interface{})

	// Warn logs a warning message
	Warn(msg string, fields ...interface{})

	// Error logs an error message
	Error(msg string, fields ...interface{})

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...interface{})

	// With returns a logger with additional fields
	With(fields ...interface{}) Logger

	// WithContext returns a logger with context
	WithContext(ctx context.Context) Logger

	// SetLevel sets the logging level
	SetLevel(level LogLevel)

	// GetLevel returns the current logging level
	GetLevel() LogLevel
}

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	// Allow returns true if the request is allowed
	Allow(ctx context.Context) bool

	// Wait waits until the request is allowed
	Wait(ctx context.Context) error

	// GetRate returns the current rate limit
	GetRate() float64

	// GetBurst returns the burst size
	GetBurst() int
}

// HTTPClient defines the HTTP client interface
type HTTPClient interface {
	// Get performs a GET request
	Get(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)

	// Post performs a POST request
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*HTTPResponse, error)

	// Put performs a PUT request
	Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*HTTPResponse, error)

	// Delete performs a DELETE request
	Delete(ctx context.Context, url string, headers map[string]string) (*HTTPResponse, error)

	// Patch performs a PATCH request
	Patch(ctx context.Context, url string, body interface{}, headers map[string]string) (*HTTPResponse, error)

	// SetTimeout sets the request timeout
	SetTimeout(timeout time.Duration)

	// SetRetryPolicy sets the retry policy
	SetRetryPolicy(policy RetryPolicy)

	// SetCircuitBreaker sets the circuit breaker
	SetCircuitBreaker(cb CircuitBreaker)

	// GetMetrics returns client metrics
	GetMetrics() HTTPMetrics
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	Duration   time.Duration       `json:"duration"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []int         `json:"retryable_errors"`
}

// CircuitBreaker defines circuit breaker behavior
type CircuitBreaker interface {
	// Execute executes a function with circuit breaker protection
	Execute(fn func() error) error

	// GetState returns the current circuit breaker state
	GetState() CircuitBreakerState

	// GetMetrics returns circuit breaker metrics
	GetMetrics() CircuitBreakerMetrics
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreakerMetrics contains circuit breaker metrics
type CircuitBreakerMetrics struct {
	Requests    int64     `json:"requests"`
	Successes   int64     `json:"successes"`
	Failures    int64     `json:"failures"`
	Timeouts    int64     `json:"timeouts"`
	LastFailure time.Time `json:"last_failure"`
	LastSuccess time.Time `json:"last_success"`
}

// HTTPMetrics contains HTTP client metrics
type HTTPMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
}

// EventBus defines the event bus interface
type EventBus interface {
	// Publish publishes an event
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes to events
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe unsubscribes from events
	Unsubscribe(eventType string, handler EventHandler) error

	// GetMetrics returns event bus metrics
	GetMetrics() EventBusMetrics
}

// Event represents an event
type Event interface {
	// GetType returns the event type
	GetType() string

	// GetData returns the event data
	GetData() interface{}

	// GetTimestamp returns the event timestamp
	GetTimestamp() time.Time

	// GetID returns the event ID
	GetID() string

	// GetSource returns the event source
	GetSource() string
}

// EventHandler handles events
type EventHandler interface {
	// Handle handles an event
	Handle(ctx context.Context, event Event) error

	// GetID returns the handler ID
	GetID() string
}

// EventBusMetrics contains event bus metrics
type EventBusMetrics struct {
	EventsPublished   int64 `json:"events_published"`
	EventsProcessed   int64 `json:"events_processed"`
	EventsFailed      int64 `json:"events_failed"`
	ActiveSubscribers int   `json:"active_subscribers"`
}

// Cache defines the cache interface
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in cache
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// Clear clears all cache entries
	Clear(ctx context.Context) error

	// GetMetrics returns cache metrics
	GetMetrics() CacheMetrics
}

// CacheMetrics contains cache metrics
type CacheMetrics struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRatio  float64 `json:"hit_ratio"`
	Size      int64   `json:"size"`
	Evictions int64   `json:"evictions"`
}

// Storage defines the storage interface
type Storage interface {
	// Read reads data from storage
	Read(ctx context.Context, key string) ([]byte, error)

	// Write writes data to storage
	Write(ctx context.Context, key string, data []byte) error

	// Delete deletes data from storage
	Delete(ctx context.Context, key string) error

	// List lists keys in storage
	List(ctx context.Context, prefix string) ([]string, error)

	// Exists checks if a key exists in storage
	Exists(ctx context.Context, key string) (bool, error)

	// GetMetrics returns storage metrics
	GetMetrics() StorageMetrics
}

// StorageMetrics contains storage metrics
type StorageMetrics struct {
	ReadOperations   int64 `json:"read_operations"`
	WriteOperations  int64 `json:"write_operations"`
	DeleteOperations int64 `json:"delete_operations"`
	TotalSize        int64 `json:"total_size"`
	ErrorCount       int64 `json:"error_count"`
}

// MetricsCollector collects and exports metrics
type MetricsCollector interface {
	// Counter creates or retrieves a counter metric
	Counter(name string, labels map[string]string) Counter

	// Gauge creates or retrieves a gauge metric
	Gauge(name string, labels map[string]string) Gauge

	// Histogram creates or retrieves a histogram metric
	Histogram(name string, labels map[string]string, buckets []float64) Histogram

	// Export exports metrics in the specified format
	Export(format string, writer io.Writer) error

	// GetMetrics returns all collected metrics
	GetMetrics() map[string]interface{}
}

// Counter represents a counter metric
type Counter interface {
	// Inc increments the counter by 1
	Inc()

	// Add adds the given value to the counter
	Add(value float64)

	// Get returns the current counter value
	Get() float64
}

// Gauge represents a gauge metric
type Gauge interface {
	// Set sets the gauge value
	Set(value float64)

	// Inc increments the gauge by 1
	Inc()

	// Dec decrements the gauge by 1
	Dec()

	// Add adds the given value to the gauge
	Add(value float64)

	// Sub subtracts the given value from the gauge
	Sub(value float64)

	// Get returns the current gauge value
	Get() float64
}

// Histogram represents a histogram metric
type Histogram interface {
	// Observe adds an observation to the histogram
	Observe(value float64)

	// GetBuckets returns the histogram buckets
	GetBuckets() map[float64]int64

	// GetCount returns the total number of observations
	GetCount() int64

	// GetSum returns the sum of all observations
	GetSum() float64
}
