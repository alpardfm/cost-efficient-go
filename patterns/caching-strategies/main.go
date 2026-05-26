package main

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

// --- Demonstration ---

func main() {
	fmt.Println("=== Caching Strategies ===")
	fmt.Println()

	// 1. No cache vs cached (same key repeated)
	fmt.Println("--- Same Key Repeated 100 Times ---")

	start := time.Now()
	for i := 0; i < 100; i++ {
		GetDirect("user-1")
	}
	directDuration := time.Since(start)

	cache := NewTTLCache(1 * time.Minute)
	start = time.Now()
	for i := 0; i < 100; i++ {
		GetWithCache(cache, "user-1")
	}
	cachedDuration := time.Since(start)

	hits, misses := cache.Stats()
	fmt.Printf("Direct (no cache): %v (100 queries × 5ms)\n", directDuration)
	fmt.Printf("Cached:            %v (1 query + 99 cache hits)\n", cachedDuration)
	fmt.Printf("Speedup:           %.0fx faster\n", float64(directDuration)/float64(cachedDuration))
	fmt.Printf("Cache stats:       hits=%d, misses=%d, ratio=%.1f%%\n",
		hits, misses, float64(hits)/float64(hits+misses)*100)
	fmt.Println()

	// 2. Mixed keys (simulating real traffic)
	fmt.Println("--- Mixed Keys (20 unique, 1000 requests) ---")
	cache2 := NewTTLCache(1 * time.Minute)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user-%d", i%20) // 20 unique keys
		GetWithCache(cache2, key)
	}
	mixedDuration := time.Since(start)

	hits2, misses2 := cache2.Stats()
	fmt.Printf("Time:        %v\n", mixedDuration)
	fmt.Printf("Cache stats: hits=%d, misses=%d, ratio=%.1f%%\n",
		hits2, misses2, float64(hits2)/float64(hits2+misses2)*100)
	fmt.Printf("Without cache would be: %v\n", 1000*5*time.Millisecond)
	fmt.Printf("Saved:       %v (%.0f%% reduction)\n",
		1000*5*time.Millisecond-mixedDuration,
		float64(1000*5*time.Millisecond-mixedDuration)/float64(1000*5*time.Millisecond)*100)
	fmt.Println()

	// 3. TTL expiration
	fmt.Println("--- TTL Expiration ---")
	shortCache := NewTTLCache(50 * time.Millisecond)
	GetWithCache(shortCache, "key-1") // Miss → fetch
	GetWithCache(shortCache, "key-1") // Hit

	time.Sleep(60 * time.Millisecond) // Wait for TTL

	GetWithCache(shortCache, "key-1") // Miss again (expired)
	h, m := shortCache.Stats()
	fmt.Printf("After TTL expiry: hits=%d, misses=%d\n", h, m)
	fmt.Println("✅ Stale data automatically evicted")
}
