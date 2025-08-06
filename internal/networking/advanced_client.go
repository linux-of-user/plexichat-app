// Package networking provides advanced networking capabilities
package networking

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"plexichat-client/internal/interfaces"
	"plexichat-client/pkg/logging"
)

// AdvancedHTTPClient implements a sophisticated HTTP client with enterprise features
type AdvancedHTTPClient struct {
	client          *http.Client
	baseURL         string
	defaultHeaders  map[string]string
	timeout         time.Duration
	retryPolicy     interfaces.RetryPolicy
	circuitBreaker  interfaces.CircuitBreaker
	connectionPool  *ConnectionPool
	loadBalancer    LoadBalancer
	metrics         *HTTPClientMetrics
	interceptors    []RequestInterceptor
	responseFilters []ResponseFilter
	logger          interfaces.Logger
	mu              sync.RWMutex
	rateLimiter     RateLimiter
	authProvider    AuthProvider
	middleware      []Middleware
}

// ConnectionPool manages HTTP connections with advanced pooling
type ConnectionPool struct {
	maxIdleConns          int
	maxIdleConnsPerHost   int
	maxConnsPerHost       int
	idleConnTimeout       time.Duration
	keepAlive             time.Duration
	tlsHandshakeTimeout   time.Duration
	expectContinueTimeout time.Duration
	transport             *http.Transport
	mu                    sync.RWMutex
	stats                 ConnectionPoolStats
}

// ConnectionPoolStats tracks connection pool statistics
type ConnectionPoolStats struct {
	ActiveConnections  int64 `json:"active_connections"`
	IdleConnections    int64 `json:"idle_connections"`
	TotalConnections   int64 `json:"total_connections"`
	ConnectionsCreated int64 `json:"connections_created"`
	ConnectionsReused  int64 `json:"connections_reused"`
	ConnectionErrors   int64 `json:"connection_errors"`
}

// LoadBalancer distributes requests across multiple endpoints
type LoadBalancer interface {
	// SelectEndpoint selects an endpoint for the request
	SelectEndpoint(ctx context.Context, request *http.Request) (string, error)

	// MarkHealthy marks an endpoint as healthy
	MarkHealthy(endpoint string)

	// MarkUnhealthy marks an endpoint as unhealthy
	MarkUnhealthy(endpoint string, err error)

	// GetEndpoints returns all configured endpoints
	GetEndpoints() []Endpoint

	// GetMetrics returns load balancer metrics
	GetMetrics() LoadBalancerMetrics
}

// Endpoint represents a backend endpoint
type Endpoint struct {
	URL      string            `json:"url"`
	Weight   int               `json:"weight"`
	Healthy  bool              `json:"healthy"`
	Metadata map[string]string `json:"metadata"`
	LastSeen time.Time         `json:"last_seen"`
}

// LoadBalancerMetrics contains load balancer metrics
type LoadBalancerMetrics struct {
	TotalRequests      int64            `json:"total_requests"`
	EndpointRequests   map[string]int64 `json:"endpoint_requests"`
	HealthyEndpoints   int              `json:"healthy_endpoints"`
	UnhealthyEndpoints int              `json:"unhealthy_endpoints"`
}

// RateLimiter controls request rate
type RateLimiter interface {
	// Allow checks if a request is allowed
	Allow(ctx context.Context) bool

	// Wait waits until a request is allowed
	Wait(ctx context.Context) error

	// GetMetrics returns rate limiter metrics
	GetMetrics() RateLimiterMetrics
}

// RateLimiterMetrics contains rate limiter metrics
type RateLimiterMetrics struct {
	RequestsAllowed int64   `json:"requests_allowed"`
	RequestsBlocked int64   `json:"requests_blocked"`
	CurrentRate     float64 `json:"current_rate"`
	BurstCapacity   int     `json:"burst_capacity"`
}

// AuthProvider handles authentication
type AuthProvider interface {
	// GetAuthHeader returns the authentication header
	GetAuthHeader(ctx context.Context) (string, string, error)

	// RefreshToken refreshes the authentication token
	RefreshToken(ctx context.Context) error

	// IsTokenValid checks if the current token is valid
	IsTokenValid(ctx context.Context) bool
}

// RequestInterceptor intercepts outgoing requests
type RequestInterceptor interface {
	// Intercept intercepts and potentially modifies a request
	Intercept(ctx context.Context, req *http.Request) error
}

// ResponseFilter filters incoming responses
type ResponseFilter interface {
	// Filter filters and potentially modifies a response
	Filter(ctx context.Context, resp *http.Response) error
}

// Middleware provides request/response middleware
type Middleware interface {
	// Process processes the request/response
	Process(ctx context.Context, req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)
}

// HTTPClientMetrics tracks HTTP client metrics
type HTTPClientMetrics struct {
	totalRequests       int64
	successRequests     int64
	failedRequests      int64
	retryAttempts       int64
	circuitBreakerTrips int64
	latencyHistogram    map[time.Duration]int64
	mu                  sync.RWMutex
}

// NewAdvancedHTTPClient creates a new advanced HTTP client
func NewAdvancedHTTPClient(config ClientConfig) *AdvancedHTTPClient {
	// Create connection pool
	pool := &ConnectionPool{
		maxIdleConns:          config.MaxIdleConns,
		maxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		maxConnsPerHost:       config.MaxConnsPerHost,
		idleConnTimeout:       config.IdleConnTimeout,
		keepAlive:             config.KeepAlive,
		tlsHandshakeTimeout:   config.TLSHandshakeTimeout,
		expectContinueTimeout: config.ExpectContinueTimeout,
	}

	// Configure transport
	transport := &http.Transport{
		MaxIdleConns:          pool.maxIdleConns,
		MaxIdleConnsPerHost:   pool.maxIdleConnsPerHost,
		MaxConnsPerHost:       pool.maxConnsPerHost,
		IdleConnTimeout:       pool.idleConnTimeout,
		TLSHandshakeTimeout:   pool.tlsHandshakeTimeout,
		ExpectContinueTimeout: pool.expectContinueTimeout,
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: pool.keepAlive,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		},
	}

	pool.transport = transport

	// Create HTTP client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	client := &AdvancedHTTPClient{
		client:         httpClient,
		baseURL:        config.BaseURL,
		defaultHeaders: config.DefaultHeaders,
		timeout:        config.Timeout,
		retryPolicy:    config.RetryPolicy,
		connectionPool: pool,
		metrics: &HTTPClientMetrics{
			latencyHistogram: make(map[time.Duration]int64),
		},
		interceptors:    make([]RequestInterceptor, 0),
		responseFilters: make([]ResponseFilter, 0),
		middleware:      make([]Middleware, 0),
		logger:          logging.GetLogger("http-client"),
	}

	// Set circuit breaker if provided
	if config.CircuitBreaker != nil {
		client.circuitBreaker = config.CircuitBreaker
	}

	// Set load balancer if provided
	if config.LoadBalancer != nil {
		client.loadBalancer = config.LoadBalancer
	}

	// Set rate limiter if provided
	if config.RateLimiter != nil {
		client.rateLimiter = config.RateLimiter
	}

	// Set auth provider if provided
	if config.AuthProvider != nil {
		client.authProvider = config.AuthProvider
	}

	return client
}

// ClientConfig contains configuration for the HTTP client
type ClientConfig struct {
	BaseURL               string                    `json:"base_url"`
	Timeout               time.Duration             `json:"timeout"`
	MaxIdleConns          int                       `json:"max_idle_conns"`
	MaxIdleConnsPerHost   int                       `json:"max_idle_conns_per_host"`
	MaxConnsPerHost       int                       `json:"max_conns_per_host"`
	IdleConnTimeout       time.Duration             `json:"idle_conn_timeout"`
	KeepAlive             time.Duration             `json:"keep_alive"`
	TLSHandshakeTimeout   time.Duration             `json:"tls_handshake_timeout"`
	ExpectContinueTimeout time.Duration             `json:"expect_continue_timeout"`
	DialTimeout           time.Duration             `json:"dial_timeout"`
	InsecureSkipVerify    bool                      `json:"insecure_skip_verify"`
	DefaultHeaders        map[string]string         `json:"default_headers"`
	RetryPolicy           interfaces.RetryPolicy    `json:"retry_policy"`
	CircuitBreaker        interfaces.CircuitBreaker `json:"-"`
	LoadBalancer          LoadBalancer              `json:"-"`
	RateLimiter           RateLimiter               `json:"-"`
	AuthProvider          AuthProvider              `json:"-"`
}

// Get performs a GET request
func (c *AdvancedHTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*interfaces.HTTPResponse, error) {
	return c.doRequest(ctx, "GET", url, nil, headers)
}

// Post performs a POST request
func (c *AdvancedHTTPClient) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	return c.doRequest(ctx, "POST", url, body, headers)
}

// Put performs a PUT request
func (c *AdvancedHTTPClient) Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	return c.doRequest(ctx, "PUT", url, body, headers)
}

// Delete performs a DELETE request
func (c *AdvancedHTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*interfaces.HTTPResponse, error) {
	return c.doRequest(ctx, "DELETE", url, nil, headers)
}

// Patch performs a PATCH request
func (c *AdvancedHTTPClient) Patch(ctx context.Context, url string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	return c.doRequest(ctx, "PATCH", url, body, headers)
}

// SetTimeout sets the request timeout
func (c *AdvancedHTTPClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = timeout
	c.client.Timeout = timeout
}

// SetRetryPolicy sets the retry policy
func (c *AdvancedHTTPClient) SetRetryPolicy(policy interfaces.RetryPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryPolicy = policy
}

// SetCircuitBreaker sets the circuit breaker
func (c *AdvancedHTTPClient) SetCircuitBreaker(cb interfaces.CircuitBreaker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.circuitBreaker = cb
}

// GetMetrics returns client metrics
func (c *AdvancedHTTPClient) GetMetrics() interfaces.HTTPMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	// Calculate average latency
	var totalLatency time.Duration
	var totalSamples int64
	for latency, count := range c.metrics.latencyHistogram {
		totalLatency += time.Duration(int64(latency) * count)
		totalSamples += count
	}

	var avgLatency time.Duration
	if totalSamples > 0 {
		avgLatency = totalLatency / time.Duration(totalSamples)
	}

	return interfaces.HTTPMetrics{
		TotalRequests:      c.metrics.totalRequests,
		SuccessfulRequests: c.metrics.successRequests,
		FailedRequests:     c.metrics.failedRequests,
		AverageLatency:     avgLatency,
		P95Latency:         c.calculatePercentile(0.95),
		P99Latency:         c.calculatePercentile(0.99),
	}
}

// AddInterceptor adds a request interceptor
func (c *AdvancedHTTPClient) AddInterceptor(interceptor RequestInterceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

// AddResponseFilter adds a response filter
func (c *AdvancedHTTPClient) AddResponseFilter(filter ResponseFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseFilters = append(c.responseFilters, filter)
}

// AddMiddleware adds middleware
func (c *AdvancedHTTPClient) AddMiddleware(middleware Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middleware = append(c.middleware, middleware)
}

// doRequest performs the actual HTTP request with all features
func (c *AdvancedHTTPClient) doRequest(ctx context.Context, method, urlPath string, body interface{}, headers map[string]string) (*interfaces.HTTPResponse, error) {
	start := time.Now()

	// Increment total requests
	atomic.AddInt64(&c.metrics.totalRequests, 1)

	// Check rate limiter
	if c.rateLimiter != nil {
		if !c.rateLimiter.Allow(ctx) {
			return nil, fmt.Errorf("request rate limited")
		}
	}

	// Build full URL
	fullURL, err := c.buildURL(urlPath)
	if err != nil {
		atomic.AddInt64(&c.metrics.failedRequests, 1)
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Use load balancer if available
	if c.loadBalancer != nil {
		endpoint, err := c.loadBalancer.SelectEndpoint(ctx, nil)
		if err != nil {
			atomic.AddInt64(&c.metrics.failedRequests, 1)
			return nil, fmt.Errorf("load balancer failed: %w", err)
		}
		fullURL = endpoint + urlPath
	}

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := c.serializeBody(body)
		if err != nil {
			atomic.AddInt64(&c.metrics.failedRequests, 1)
			return nil, fmt.Errorf("failed to serialize body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		atomic.AddInt64(&c.metrics.failedRequests, 1)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setHeaders(req, headers)

	// Add authentication if available
	if c.authProvider != nil {
		if !c.authProvider.IsTokenValid(ctx) {
			if err := c.authProvider.RefreshToken(ctx); err != nil {
				c.logger.Warn("Failed to refresh token", "error", err)
			}
		}

		headerName, headerValue, err := c.authProvider.GetAuthHeader(ctx)
		if err == nil {
			req.Header.Set(headerName, headerValue)
		}
	}

	// Apply interceptors
	for _, interceptor := range c.interceptors {
		if err := interceptor.Intercept(ctx, req); err != nil {
			atomic.AddInt64(&c.metrics.failedRequests, 1)
			return nil, fmt.Errorf("interceptor failed: %w", err)
		}
	}

	// Execute request with retry and circuit breaker
	var resp *http.Response
	if c.circuitBreaker != nil {
		err = c.circuitBreaker.Execute(func() error {
			resp, err = c.executeWithRetry(ctx, req)
			return err
		})
	} else {
		resp, err = c.executeWithRetry(ctx, req)
	}

	if err != nil {
		atomic.AddInt64(&c.metrics.failedRequests, 1)
		return nil, err
	}

	defer resp.Body.Close()

	// Apply response filters
	for _, filter := range c.responseFilters {
		if err := filter.Filter(ctx, resp); err != nil {
			atomic.AddInt64(&c.metrics.failedRequests, 1)
			return nil, fmt.Errorf("response filter failed: %w", err)
		}
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		atomic.AddInt64(&c.metrics.failedRequests, 1)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	duration := time.Since(start)

	// Update metrics
	atomic.AddInt64(&c.metrics.successRequests, 1)
	c.updateLatencyMetrics(duration)

	// Mark endpoint as healthy if using load balancer
	if c.loadBalancer != nil {
		c.loadBalancer.MarkHealthy(fullURL)
	}

	return &interfaces.HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
		Duration:   duration,
	}, nil
}

// executeWithRetry executes the request with retry logic
func (c *AdvancedHTTPClient) executeWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(c.retryPolicy.InitialDelay) *
				float64(attempt) * c.retryPolicy.BackoffFactor)
			if delay > c.retryPolicy.MaxDelay {
				delay = c.retryPolicy.MaxDelay
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			atomic.AddInt64(&c.metrics.retryAttempts, 1)
		}

		// Clone request for retry
		reqClone := req.Clone(ctx)

		// Execute middleware chain
		resp, err := c.executeMiddleware(ctx, reqClone)
		if err == nil && c.isSuccessStatusCode(resp.StatusCode) {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryableError(err, resp) {
			break
		}

		c.logger.Debug("Request failed, retrying",
			"attempt", attempt+1,
			"max_retries", c.retryPolicy.MaxRetries,
			"error", err)
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w",
		c.retryPolicy.MaxRetries+1, lastErr)
}

// executeMiddleware executes the middleware chain
func (c *AdvancedHTTPClient) executeMiddleware(ctx context.Context, req *http.Request) (*http.Response, error) {
	if len(c.middleware) == 0 {
		return c.client.Do(req)
	}

	// Create middleware chain
	var handler func(*http.Request) (*http.Response, error)
	handler = c.client.Do

	// Apply middleware in reverse order
	for i := len(c.middleware) - 1; i >= 0; i-- {
		middleware := c.middleware[i]
		nextHandler := handler
		handler = func(r *http.Request) (*http.Response, error) {
			return middleware.Process(ctx, r, nextHandler)
		}
	}

	return handler(req)
}

// buildURL builds the full URL from base URL and path
func (c *AdvancedHTTPClient) buildURL(path string) (string, error) {
	if c.baseURL == "" {
		return path, nil
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(rel).String(), nil
}

// serializeBody serializes the request body
func (c *AdvancedHTTPClient) serializeBody(body interface{}) ([]byte, error) {
	switch v := body.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case io.Reader:
		return io.ReadAll(v)
	default:
		return json.Marshal(v)
	}
}

// setHeaders sets request headers
func (c *AdvancedHTTPClient) setHeaders(req *http.Request, headers map[string]string) {
	// Set default headers
	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}

	// Set request-specific headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set content type if not specified and body exists
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
}

// updateLatencyMetrics updates latency metrics
func (c *AdvancedHTTPClient) updateLatencyMetrics(duration time.Duration) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	// Round to nearest millisecond for bucketing
	bucket := duration.Truncate(time.Millisecond)
	c.metrics.latencyHistogram[bucket]++
}

// calculatePercentile calculates the given percentile from latency histogram
func (c *AdvancedHTTPClient) calculatePercentile(percentile float64) time.Duration {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	if len(c.metrics.latencyHistogram) == 0 {
		return 0
	}

	// Collect all samples
	var samples []time.Duration
	for latency, count := range c.metrics.latencyHistogram {
		for i := int64(0); i < count; i++ {
			samples = append(samples, latency)
		}
	}

	if len(samples) == 0 {
		return 0
	}

	// Sort samples
	for i := 0; i < len(samples); i++ {
		for j := i + 1; j < len(samples); j++ {
			if samples[i] > samples[j] {
				samples[i], samples[j] = samples[j], samples[i]
			}
		}
	}

	// Calculate percentile index
	index := int(float64(len(samples)) * percentile)
	if index >= len(samples) {
		index = len(samples) - 1
	}

	return samples[index]
}

// isSuccessStatusCode checks if the status code indicates success
func (c *AdvancedHTTPClient) isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// isRetryableError checks if an error or response is retryable
func (c *AdvancedHTTPClient) isRetryableError(err error, resp *http.Response) bool {
	// Network errors are generally retryable
	if err != nil {
		// Check for timeout errors
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}

		// Check for connection errors
		if _, ok := err.(*net.OpError); ok {
			return true
		}

		return false
	}

	// Check if status code is retryable
	if resp != nil {
		for _, retryableCode := range c.retryPolicy.RetryableErrors {
			if resp.StatusCode == retryableCode {
				return true
			}
		}

		// Default retryable status codes
		switch resp.StatusCode {
		case 408, 429, 500, 502, 503, 504:
			return true
		}
	}

	return false
}

// GetConnectionPoolStats returns connection pool statistics
func (c *AdvancedHTTPClient) GetConnectionPoolStats() ConnectionPoolStats {
	c.connectionPool.mu.RLock()
	defer c.connectionPool.mu.RUnlock()
	return c.connectionPool.stats
}

// Close closes the HTTP client and cleans up resources
func (c *AdvancedHTTPClient) Close() error {
	if c.client != nil && c.client.Transport != nil {
		if transport, ok := c.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}
