package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"plexichat-client/pkg/client"
	"plexichat-client/pkg/logging"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache provides intelligent caching for API responses
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	ttl      map[string]time.Duration
	maxSize  int
	logger   *logging.Logger
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxSize         int                       `json:"max_size"`
	DefaultTTL      time.Duration             `json:"default_ttl"`
	TypeSpecificTTL map[string]time.Duration  `json:"type_specific_ttl"`
	Enabled         bool                      `json:"enabled"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:    1000,
		DefaultTTL: 5 * time.Minute,
		TypeSpecificTTL: map[string]time.Duration{
			"users":    30 * time.Minute, // Users don't change often
			"rooms":    15 * time.Minute, // Rooms are relatively stable
			"messages": 2 * time.Minute,  // Messages can be updated/deleted
			"files":    10 * time.Minute, // File metadata is stable
			"health":   30 * time.Second, // Health status changes frequently
		},
		Enabled: true,
	}
}

// NewCache creates a new cache instance
func NewCache(config *CacheConfig) *Cache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     config.TypeSpecificTTL,
		maxSize: config.MaxSize,
		logger:  logging.NewLogger(logging.DEBUG, nil, true),
	}
}

// generateKey creates a cache key from type and identifier
func (c *Cache) generateKey(cacheType, identifier string) string {
	return fmt.Sprintf("%s:%s", cacheType, identifier)
}

// Get retrieves an item from cache
func (c *Cache) Get(cacheType, identifier string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey(cacheType, identifier)
	entry, exists := c.entries[key]
	
	if !exists {
		c.logger.Debug("Cache miss: %s", key)
		return nil, false
	}

	if entry.IsExpired() {
		c.logger.Debug("Cache expired: %s", key)
		// Don't delete here to avoid write lock, let cleanup handle it
		return nil, false
	}

	c.logger.Debug("Cache hit: %s", key)
	return entry.Data, true
}

// Set stores an item in cache
func (c *Cache) Set(cacheType, identifier string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	key := c.generateKey(cacheType, identifier)
	ttl := c.getTTL(cacheType)
	
	entry := &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	c.entries[key] = entry
	c.logger.Debug("Cache set: %s (TTL: %v)", key, ttl)
}

// Delete removes an item from cache
func (c *Cache) Delete(cacheType, identifier string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(cacheType, identifier)
	delete(c.entries, key)
	c.logger.Debug("Cache delete: %s", key)
}

// Clear removes all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.logger.Debug("Cache cleared")
}

// ClearType removes all items of a specific type
func (c *Cache) ClearType(cacheType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := cacheType + ":"
	for key := range c.entries {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
	c.logger.Debug("Cache cleared for type: %s", cacheType)
}

// getTTL returns the TTL for a specific cache type
func (c *Cache) getTTL(cacheType string) time.Duration {
	if ttl, exists := c.ttl[cacheType]; exists {
		return ttl
	}
	return 5 * time.Minute // Default TTL
}

// evictOldest removes the oldest entry from cache
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.logger.Debug("Cache evicted oldest: %s", oldestKey)
	}
}

// Cleanup removes expired entries
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiredKeys []string
	for key, entry := range c.entries {
		if entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.entries, key)
	}

	if len(expiredKeys) > 0 {
		c.logger.Debug("Cache cleanup removed %d expired entries", len(expiredKeys))
	}
}

// StartCleanupRoutine starts a background cleanup routine
func (c *Cache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.Cleanup()
		}
	}()
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"total_entries": len(c.entries),
		"max_size":      c.maxSize,
	}

	// Count by type
	typeCounts := make(map[string]int)
	expiredCount := 0
	
	for key, entry := range c.entries {
		// Extract type from key
		if colonIndex := len(key); colonIndex > 0 {
			for i, char := range key {
				if char == ':' {
					cacheType := key[:i]
					typeCounts[cacheType]++
					break
				}
			}
		}
		
		if entry.IsExpired() {
			expiredCount++
		}
	}

	stats["by_type"] = typeCounts
	stats["expired_entries"] = expiredCount

	return stats
}

// CachedClient wraps the API client with caching
type CachedClient struct {
	*client.Client
	cache   *Cache
	enabled bool
}

// NewCachedClient creates a new cached client
func NewCachedClient(apiClient *client.Client, config *CacheConfig) *CachedClient {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache := NewCache(config)
	
	// Start cleanup routine
	cache.StartCleanupRoutine(1 * time.Minute)

	return &CachedClient{
		Client:  apiClient,
		cache:   cache,
		enabled: config.Enabled,
	}
}

// GetUser gets user with caching
func (c *CachedClient) GetUser(userID string) (*client.UserResponse, error) {
	if c.enabled {
		if cached, found := c.cache.Get("users", userID); found {
			if user, ok := cached.(*client.UserResponse); ok {
				return user, nil
			}
		}
	}

	// Fetch from API
	user, err := c.Client.GetUser(nil, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if c.enabled && user != nil {
		c.cache.Set("users", userID, user)
	}

	return user, nil
}

// GetMessages gets messages with caching
func (c *CachedClient) GetMessages(otherUserID string, limit, page int) (*client.MessageListResponse, error) {
	cacheKey := fmt.Sprintf("%s:%d:%d", otherUserID, limit, page)
	
	if c.enabled {
		if cached, found := c.cache.Get("messages", cacheKey); found {
			if messages, ok := cached.(*client.MessageListResponse); ok {
				return messages, nil
			}
		}
	}

	// Fetch from API
	messages, err := c.Client.GetMessages(nil, otherUserID, limit, page)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if c.enabled && messages != nil {
		c.cache.Set("messages", cacheKey, messages)
	}

	return messages, nil
}

// InvalidateUser removes user from cache
func (c *CachedClient) InvalidateUser(userID string) {
	if c.enabled {
		c.cache.Delete("users", userID)
	}
}

// InvalidateMessages removes messages from cache
func (c *CachedClient) InvalidateMessages(otherUserID string) {
	if c.enabled {
		// Clear all message pages for this conversation
		c.cache.ClearType("messages")
	}
}

// GetCacheStats returns cache statistics
func (c *CachedClient) GetCacheStats() map[string]interface{} {
	if !c.enabled {
		return map[string]interface{}{"enabled": false}
	}
	
	stats := c.cache.Stats()
	stats["enabled"] = true
	return stats
}
