// Package networking provides rate limiting implementation
package networking

import (
	"context"
	"sync"
	"time"
)

// TokenBucketRateLimiter implements a token bucket rate limiter
type TokenBucketRateLimiter struct {
	rate       float64
	capacity   int
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
	metrics    RateLimiterMetrics
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(rate float64, capacity int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastRefill: time.Now(),
		metrics: RateLimiterMetrics{
			BurstCapacity: capacity,
		},
	}
}

// Allow checks if a request is allowed
func (rl *TokenBucketRateLimiter) Allow(ctx context.Context) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= 1.0 {
		rl.tokens--
		rl.metrics.RequestsAllowed++
		rl.metrics.CurrentRate = rl.rate
		return true
	}

	rl.metrics.RequestsBlocked++
	return false
}

// Wait waits until a request is allowed
func (rl *TokenBucketRateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow(ctx) {
			return nil
		}

		// Calculate wait time
		rl.mu.Lock()
		waitTime := time.Duration((1.0 - rl.tokens) / rl.rate * float64(time.Second))
		rl.mu.Unlock()

		if waitTime > time.Second {
			waitTime = time.Second
		}

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetMetrics returns rate limiter metrics
func (rl *TokenBucketRateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.metrics
}

// refill refills the token bucket
func (rl *TokenBucketRateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now

	tokensToAdd := elapsed * rl.rate
	rl.tokens += tokensToAdd

	if rl.tokens > float64(rl.capacity) {
		rl.tokens = float64(rl.capacity)
	}
}

// SlidingWindowRateLimiter implements a sliding window rate limiter
type SlidingWindowRateLimiter struct {
	limit      int
	window     time.Duration
	requests   []time.Time
	mu         sync.Mutex
	metrics    RateLimiterMetrics
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0),
		metrics: RateLimiterMetrics{
			BurstCapacity: limit,
		},
	}
}

// Allow checks if a request is allowed
func (rl *SlidingWindowRateLimiter) Allow(ctx context.Context) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.cleanOldRequests(now)

	if len(rl.requests) < rl.limit {
		rl.requests = append(rl.requests, now)
		rl.metrics.RequestsAllowed++
		rl.metrics.CurrentRate = float64(len(rl.requests)) / rl.window.Seconds()
		return true
	}

	rl.metrics.RequestsBlocked++
	return false
}

// Wait waits until a request is allowed
func (rl *SlidingWindowRateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow(ctx) {
			return nil
		}

		// Calculate wait time based on oldest request
		rl.mu.Lock()
		var waitTime time.Duration
		if len(rl.requests) > 0 {
			waitTime = rl.requests[0].Add(rl.window).Sub(time.Now())
		} else {
			waitTime = 100 * time.Millisecond
		}
		rl.mu.Unlock()

		if waitTime <= 0 {
			waitTime = 100 * time.Millisecond
		}

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetMetrics returns rate limiter metrics
func (rl *SlidingWindowRateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.metrics
}

// cleanOldRequests removes requests outside the sliding window
func (rl *SlidingWindowRateLimiter) cleanOldRequests(now time.Time) {
	cutoff := now.Add(-rl.window)
	
	// Find first request within window
	start := 0
	for i, req := range rl.requests {
		if req.After(cutoff) {
			start = i
			break
		}
		start = i + 1
	}

	// Remove old requests
	if start > 0 {
		rl.requests = rl.requests[start:]
	}
}

// AdaptiveRateLimiter implements an adaptive rate limiter that adjusts based on system load
type AdaptiveRateLimiter struct {
	baseRate        float64
	currentRate     float64
	maxRate         float64
	minRate         float64
	capacity        int
	tokens          float64
	lastRefill      time.Time
	lastAdjustment  time.Time
	adjustInterval  time.Duration
	loadThreshold   float64
	loadFunc        func() float64
	mu              sync.Mutex
	metrics         RateLimiterMetrics
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(baseRate, maxRate, minRate float64, capacity int, loadFunc func() float64) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseRate:       baseRate,
		currentRate:    baseRate,
		maxRate:        maxRate,
		minRate:        minRate,
		capacity:       capacity,
		tokens:         float64(capacity),
		lastRefill:     time.Now(),
		lastAdjustment: time.Now(),
		adjustInterval: 10 * time.Second,
		loadThreshold:  0.8,
		loadFunc:       loadFunc,
		metrics: RateLimiterMetrics{
			BurstCapacity: capacity,
		},
	}
}

// Allow checks if a request is allowed
func (rl *AdaptiveRateLimiter) Allow(ctx context.Context) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.adjustRate()
	rl.refill()

	if rl.tokens >= 1.0 {
		rl.tokens--
		rl.metrics.RequestsAllowed++
		rl.metrics.CurrentRate = rl.currentRate
		return true
	}

	rl.metrics.RequestsBlocked++
	return false
}

// Wait waits until a request is allowed
func (rl *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow(ctx) {
			return nil
		}

		// Calculate wait time
		rl.mu.Lock()
		waitTime := time.Duration((1.0 - rl.tokens) / rl.currentRate * float64(time.Second))
		rl.mu.Unlock()

		if waitTime > time.Second {
			waitTime = time.Second
		}

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetMetrics returns rate limiter metrics
func (rl *AdaptiveRateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.metrics
}

// adjustRate adjusts the rate based on system load
func (rl *AdaptiveRateLimiter) adjustRate() {
	now := time.Now()
	if now.Sub(rl.lastAdjustment) < rl.adjustInterval {
		return
	}

	rl.lastAdjustment = now

	if rl.loadFunc == nil {
		return
	}

	load := rl.loadFunc()
	
	if load > rl.loadThreshold {
		// High load, decrease rate
		rl.currentRate = rl.currentRate * 0.9
		if rl.currentRate < rl.minRate {
			rl.currentRate = rl.minRate
		}
	} else if load < rl.loadThreshold*0.5 {
		// Low load, increase rate
		rl.currentRate = rl.currentRate * 1.1
		if rl.currentRate > rl.maxRate {
			rl.currentRate = rl.maxRate
		}
	}
}

// refill refills the token bucket
func (rl *AdaptiveRateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now

	tokensToAdd := elapsed * rl.currentRate
	rl.tokens += tokensToAdd

	if rl.tokens > float64(rl.capacity) {
		rl.tokens = float64(rl.capacity)
	}
}

// CompositeRateLimiter combines multiple rate limiters
type CompositeRateLimiter struct {
	limiters []RateLimiter
	metrics  RateLimiterMetrics
	mu       sync.RWMutex
}

// NewCompositeRateLimiter creates a new composite rate limiter
func NewCompositeRateLimiter(limiters ...RateLimiter) *CompositeRateLimiter {
	return &CompositeRateLimiter{
		limiters: limiters,
	}
}

// Allow checks if a request is allowed by all limiters
func (rl *CompositeRateLimiter) Allow(ctx context.Context) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for _, limiter := range rl.limiters {
		if !limiter.Allow(ctx) {
			rl.metrics.RequestsBlocked++
			return false
		}
	}

	rl.metrics.RequestsAllowed++
	return true
}

// Wait waits until a request is allowed by all limiters
func (rl *CompositeRateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow(ctx) {
			return nil
		}

		select {
		case <-time.After(100 * time.Millisecond):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetMetrics returns composite rate limiter metrics
func (rl *CompositeRateLimiter) GetMetrics() RateLimiterMetrics {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.metrics
}

// AddLimiter adds a rate limiter to the composite
func (rl *CompositeRateLimiter) AddLimiter(limiter RateLimiter) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limiters = append(rl.limiters, limiter)
}

// RemoveLimiter removes a rate limiter from the composite
func (rl *CompositeRateLimiter) RemoveLimiter(index int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if index >= 0 && index < len(rl.limiters) {
		rl.limiters = append(rl.limiters[:index], rl.limiters[index+1:]...)
	}
}
