// Package cache provides cache interface and implementation
// for storing and retrieving key-value data, supports expiration and auto-cleanup
package cache

import (
	"sync"
	"time"
)

// Cache defines the basic operations interface for cache
// this interface allows storing any type of value and supports expiration
type Cache interface {
	// Get get cached value for specified key
	// if key exists and not expired, return value and true
	// if key doesn't exist or expired, return nil and false
	Get(key string) (interface{}, bool)

	// Set set cache value
	// use default expiration
	Set(key string, value interface{})

	// SetWithExpiration set cache value with specified expiration
	// if expiration is 0, use default
	// if negative, never expire
	SetWithExpiration(key string, value interface{}, d time.Duration)

	// Delete delete cache for specified key
	Delete(key string)

	// Clear clear all cache
	Clear()

	// Count return item count in cache
	Count() int

	// Close close cache, release resources
	// should be called when cache is no longer used
	Close()
}

// cacheItem represents a cache item
type cacheItem struct {
	value      interface{} // stored value
	expiration time.Time   // expiration
	created    time.Time   // created
}

// MemoryCache is the memory implementation of Cache interface
// stores data in memory, supports auto-expiration and periodic cleanup
type MemoryCache struct {
	defaultExpiration time.Duration        // default expiration
	cleanupInterval   time.Duration        // cleanup interval
	items             map[string]cacheItem // cache item storage
	mu                sync.RWMutex         // read-write lock for concurrency safety
	stopCleanup       chan struct{}        // stop cleanup channel
	closed            bool                 // whether cache is closed
}

// NewMemoryCache create a new memory cache
// Parameters:
//   - defaultExpiration: default cache item expiration
//   - cleanupInterval: auto-cleanup interval for expired items
//
// if cleanupInterval is 0, no auto-cleanup
// if defaultExpiration is 0, use 1 hour as default
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	if defaultExpiration <= 0 {
		defaultExpiration = time.Hour
	}

	cache := &MemoryCache{
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
		items:             make(map[string]cacheItem),
		stopCleanup:       make(chan struct{}),
	}

	// if cleanup interval set, start auto-cleanup
	if cleanupInterval > 0 {
		go cache.startCleanupTimer()
	}

	return cache
}

// Get get cached value
// if key doesn't exist or expired, return nil and false
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	// check if expired
	if !item.expiration.IsZero() && item.expiration.Before(time.Now()) {
		return nil, false
	}

	return item.value, true
}

// Set set cache value with default expiration
func (c *MemoryCache) Set(key string, value interface{}) {
	c.SetWithExpiration(key, value, c.defaultExpiration)
}

// SetWithExpiration set cache value with specified expiration
// if d is 0, use default expiration
// if d is negative, never expire
func (c *MemoryCache) SetWithExpiration(key string, value interface{}, d time.Duration) {
	var expiration time.Time

	if d == 0 {
		d = c.defaultExpiration
	}

	// if duration is negative, never expire
	if d > 0 {
		expiration = time.Now().Add(d)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:      value,
		expiration: expiration,
		created:    time.Now(),
	}
}

// Delete delete specified key from cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear clear all cache items
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
}

// Count return item count in cache
func (c *MemoryCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Close close cache, stop auto-cleanup
func (c *MemoryCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		close(c.stopCleanup)
		c.closed = true
	}
}

// startCleanupTimer start timer for periodic cleanup of expired items
func (c *MemoryCache) startCleanupTimer() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// deleteExpired delete all expired cache items
func (c *MemoryCache) deleteExpired() {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, item := range c.items {
		// if expiration set and expired, delete
		if !item.expiration.IsZero() && item.expiration.Before(now) {
			delete(c.items, k)
		}
	}
}
