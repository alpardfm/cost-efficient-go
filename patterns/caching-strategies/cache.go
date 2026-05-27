package caching_strategies

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================
// PATTERN 10: Caching Strategies
// ============================================================
// Problem: Repeated expensive computations or DB queries waste
// CPU and increase latency. In-memory caching eliminates
// redundant work.
//
// This pattern demonstrates:
// 1. Simple in-memory cache with TTL
// 2. Cache hit/miss ratio impact
// 3. sync.Map vs mutex-based cache
// 4. Cache invalidation strategies
// ============================================================

// --- Simulated Expensive Operation ---

// ExpensiveQuery simulates a database query (5ms latency).
func ExpensiveQuery(key string) string {
	time.Sleep(5 * time.Millisecond) // Simulate DB round-trip
	return fmt.Sprintf("result-for-%s", key)
}

// --- Simple TTL Cache ---

// CacheEntry holds a cached value with expiration.
type CacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

// TTLCache is a simple thread-safe cache with TTL.
type TTLCache struct {
	mu     sync.RWMutex
	items  map[string]CacheEntry
	ttl    time.Duration
	hits   int64
	misses int64
}

// NewTTLCache creates a cache with the given TTL.
func NewTTLCache(ttl time.Duration) *TTLCache {
	return &TTLCache{
		items: make(map[string]CacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a value from cache. Returns empty string and false on miss.
func (c *TTLCache) Get(key string) (string, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return "", false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
	return entry.Value, true
}

// Set stores a value in cache with TTL.
func (c *TTLCache) Set(key, value string) {
	c.mu.Lock()
	c.items[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Stats returns hit/miss counts.
func (c *TTLCache) Stats() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

// --- Cache-Aside Pattern ---

// GetWithCache implements cache-aside: check cache first, fallback to source.
func GetWithCache(cache *TTLCache, key string) string {
	if value, ok := cache.Get(key); ok {
		return value // Cache hit
	}

	// Cache miss — fetch from source
	value := ExpensiveQuery(key)
	cache.Set(key, value)
	return value
}

// --- No Cache (Direct Query) ---

// GetDirect always queries the source.
func GetDirect(key string) string {
	return ExpensiveQuery(key)
}
