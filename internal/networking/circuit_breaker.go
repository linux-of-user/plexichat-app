// Package networking provides circuit breaker implementation
package networking

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"plexichat-client/internal/interfaces"
)

// CircuitBreakerImpl implements the CircuitBreaker interface
type CircuitBreakerImpl struct {
	name             string
	maxRequests      uint32
	interval         time.Duration
	timeout          time.Duration
	failureThreshold uint32
	successThreshold uint32
	onStateChange    func(name string, from, to interfaces.CircuitBreakerState)

	mu         sync.RWMutex
	state      interfaces.CircuitBreakerState
	generation uint64
	counts     *Counts
	expiry     time.Time
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreakerConfig contains configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name             string                                                     `json:"name"`
	MaxRequests      uint32                                                     `json:"max_requests"`
	Interval         time.Duration                                              `json:"interval"`
	Timeout          time.Duration                                              `json:"timeout"`
	FailureThreshold uint32                                                     `json:"failure_threshold"`
	SuccessThreshold uint32                                                     `json:"success_threshold"`
	OnStateChange    func(name string, from, to interfaces.CircuitBreakerState) `json:"-"`
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreakerImpl {
	cb := &CircuitBreakerImpl{
		name:             config.Name,
		maxRequests:      config.MaxRequests,
		interval:         config.Interval,
		timeout:          config.Timeout,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		onStateChange:    config.OnStateChange,
		state:            interfaces.CircuitBreakerClosed,
		counts:           &Counts{},
	}

	// Set defaults
	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval == 0 {
		cb.interval = 60 * time.Second
	}
	if cb.timeout == 0 {
		cb.timeout = 60 * time.Second
	}
	if cb.failureThreshold == 0 {
		cb.failureThreshold = 5
	}
	if cb.successThreshold == 0 {
		cb.successThreshold = 1
	}

	return cb
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreakerImpl) Execute(fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	err = fn()
	cb.afterRequest(generation, err == nil)
	return err
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreakerImpl) GetState() interfaces.CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreakerImpl) GetMetrics() interfaces.CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return interfaces.CircuitBreakerMetrics{
		Requests:    int64(cb.counts.Requests),
		Successes:   int64(cb.counts.TotalSuccesses),
		Failures:    int64(cb.counts.TotalFailures),
		Timeouts:    0,           // TODO: Track timeouts separately
		LastFailure: time.Time{}, // TODO: Track last failure time
		LastSuccess: time.Time{}, // TODO: Track last success time
	}
}

// beforeRequest is called before executing the request
func (cb *CircuitBreakerImpl) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == interfaces.CircuitBreakerOpen {
		return generation, fmt.Errorf("circuit breaker is open")
	} else if state == interfaces.CircuitBreakerHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, fmt.Errorf("circuit breaker is half-open and max requests exceeded")
	}

	cb.counts.Requests++
	return generation, nil
}

// afterRequest is called after executing the request
func (cb *CircuitBreakerImpl) afterRequest(before uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess handles successful requests
func (cb *CircuitBreakerImpl) onSuccess(state interfaces.CircuitBreakerState, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == interfaces.CircuitBreakerHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.successThreshold {
		cb.setState(interfaces.CircuitBreakerClosed, now)
	}
}

// onFailure handles failed requests
func (cb *CircuitBreakerImpl) onFailure(state interfaces.CircuitBreakerState, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if state == interfaces.CircuitBreakerClosed && cb.counts.ConsecutiveFailures >= cb.failureThreshold {
		cb.setState(interfaces.CircuitBreakerOpen, now)
	} else if state == interfaces.CircuitBreakerHalfOpen {
		cb.setState(interfaces.CircuitBreakerOpen, now)
	}
}

// currentState returns the current state and generation
func (cb *CircuitBreakerImpl) currentState(now time.Time) (interfaces.CircuitBreakerState, uint64) {
	switch cb.state {
	case interfaces.CircuitBreakerClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case interfaces.CircuitBreakerOpen:
		if cb.expiry.Before(now) {
			cb.setState(interfaces.CircuitBreakerHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState changes the circuit breaker state
func (cb *CircuitBreakerImpl) setState(state interfaces.CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration creates a new generation
func (cb *CircuitBreakerImpl) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = &Counts{}

	var zero time.Time
	switch cb.state {
	case interfaces.CircuitBreakerClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case interfaces.CircuitBreakerOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // half-open
		cb.expiry = zero
	}
}

// RoundRobinLoadBalancer implements a round-robin load balancer
type RoundRobinLoadBalancer struct {
	endpoints []Endpoint
	current   uint64
	mu        sync.RWMutex
	metrics   LoadBalancerMetrics
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer(endpoints []Endpoint) *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		endpoints: endpoints,
		metrics: LoadBalancerMetrics{
			EndpointRequests: make(map[string]int64),
		},
	}
}

// SelectEndpoint selects an endpoint using round-robin algorithm
func (lb *RoundRobinLoadBalancer) SelectEndpoint(ctx context.Context, request *http.Request) (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.endpoints) == 0 {
		return "", fmt.Errorf("no endpoints available")
	}

	// Filter healthy endpoints
	var healthyEndpoints []Endpoint
	for _, endpoint := range lb.endpoints {
		if endpoint.Healthy {
			healthyEndpoints = append(healthyEndpoints, endpoint)
		}
	}

	if len(healthyEndpoints) == 0 {
		return "", fmt.Errorf("no healthy endpoints available")
	}

	// Select endpoint using round-robin
	index := atomic.AddUint64(&lb.current, 1) % uint64(len(healthyEndpoints))
	selected := healthyEndpoints[index]

	// Update metrics
	lb.metrics.TotalRequests++
	lb.metrics.EndpointRequests[selected.URL]++

	return selected.URL, nil
}

// MarkHealthy marks an endpoint as healthy
func (lb *RoundRobinLoadBalancer) MarkHealthy(endpoint string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := range lb.endpoints {
		if lb.endpoints[i].URL == endpoint {
			lb.endpoints[i].Healthy = true
			lb.endpoints[i].LastSeen = time.Now()
			break
		}
	}
}

// MarkUnhealthy marks an endpoint as unhealthy
func (lb *RoundRobinLoadBalancer) MarkUnhealthy(endpoint string, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := range lb.endpoints {
		if lb.endpoints[i].URL == endpoint {
			lb.endpoints[i].Healthy = false
			break
		}
	}
}

// GetEndpoints returns all configured endpoints
func (lb *RoundRobinLoadBalancer) GetEndpoints() []Endpoint {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	endpoints := make([]Endpoint, len(lb.endpoints))
	copy(endpoints, lb.endpoints)
	return endpoints
}

// GetMetrics returns load balancer metrics
func (lb *RoundRobinLoadBalancer) GetMetrics() LoadBalancerMetrics {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Count healthy/unhealthy endpoints
	healthy := 0
	unhealthy := 0
	for _, endpoint := range lb.endpoints {
		if endpoint.Healthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	metrics := lb.metrics
	metrics.HealthyEndpoints = healthy
	metrics.UnhealthyEndpoints = unhealthy

	return metrics
}

// AddEndpoint adds a new endpoint
func (lb *RoundRobinLoadBalancer) AddEndpoint(endpoint Endpoint) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.endpoints = append(lb.endpoints, endpoint)
	lb.metrics.EndpointRequests[endpoint.URL] = 0
}

// RemoveEndpoint removes an endpoint
func (lb *RoundRobinLoadBalancer) RemoveEndpoint(url string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, endpoint := range lb.endpoints {
		if endpoint.URL == url {
			lb.endpoints = append(lb.endpoints[:i], lb.endpoints[i+1:]...)
			delete(lb.metrics.EndpointRequests, url)
			break
		}
	}
}
