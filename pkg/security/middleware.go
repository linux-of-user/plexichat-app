package security

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	RateLimitRequests int           // Requests per minute
	RateLimitBurst    int           // Burst capacity
	JWTSecret         string        // JWT signing secret
	APIKeyHeader      string        // API key header name
	RequireHTTPS      bool          // Force HTTPS
	MaxRequestSize    int64         // Max request body size
	AllowedOrigins    []string      // CORS allowed origins
	SessionTimeout    time.Duration // Session timeout
}

// DefaultSecurityConfig returns secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		RateLimitRequests: 100, // 100 requests per minute
		RateLimitBurst:    10,  // Allow burst of 10
		APIKeyHeader:      "X-API-Key",
		RequireHTTPS:      true,
		MaxRequestSize:    10 << 20, // 10MB max request
		AllowedOrigins:    []string{"https://app.plexichat.com"},
		SessionTimeout:    24 * time.Hour,
	}
}

// RateLimiter implements per-IP rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerMinute) / 60, // Convert to per-second
		burst:    burst,
	}
}

// GetLimiter returns rate limiter for IP
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		if limiter, exists = rl.limiters[ip]; !exists {
			limiter = rate.NewLimiter(rl.rate, rl.burst)
			rl.limiters[ip] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

// CleanupOldLimiters removes inactive limiters (call periodically)
func (rl *RateLimiter) CleanupOldLimiters() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, limiter := range rl.limiters {
		// Remove limiters that haven't been used recently
		if limiter.TokensAt(time.Now()) == float64(rl.burst) {
			delete(rl.limiters, ip)
		}
	}
}

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	config      *SecurityConfig
	rateLimiter *RateLimiter
}

// NewSecurityMiddleware creates security middleware
func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityMiddleware{
		config:      config,
		rateLimiter: NewRateLimiter(config.RateLimitRequests, config.RateLimitBurst),
	}
}

// RateLimitMiddleware implements rate limiting
func (sm *SecurityMiddleware) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		limiter := sm.rateLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers
func (sm *SecurityMiddleware) SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if sm.config.RequireHTTPS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' wss: ws:"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}

// HTTPSRedirectMiddleware redirects HTTP to HTTPS
func (sm *SecurityMiddleware) HTTPSRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sm.config.RequireHTTPS && r.Header.Get("X-Forwarded-Proto") != "https" && r.TLS == nil {
			httpsURL := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func (sm *SecurityMiddleware) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range sm.config.AllowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimitMiddleware limits request body size
func (sm *SecurityMiddleware) RequestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > sm.config.MaxRequestSize {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Limit reader to prevent memory exhaustion
		r.Body = http.MaxBytesReader(w, r.Body, sm.config.MaxRequestSize)

		next.ServeHTTP(w, r)
	})
}

// GetClientIP extracts client IP from request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header (nginx)
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	return ip
}

// ValidateAPIKey validates API key using constant-time comparison
func ValidateAPIKey(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
	// Remove null bytes and control characters
	sanitized := strings.ReplaceAll(input, "\x00", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")

	// Limit length to prevent DoS
	if len(sanitized) > 10000 {
		sanitized = sanitized[:10000]
	}

	return strings.TrimSpace(sanitized)
}

// LogSecurityEvent logs security-related events
func LogSecurityEvent(eventType, clientIP, userAgent, details string) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	fmt.Printf("[SECURITY] %s | %s | IP: %s | UA: %s | %s\n",
		timestamp, eventType, clientIP, userAgent, details)
}
