package caching_strategies

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ============================================================
// Benchmarks: Cache vs No Cache, Different Hit Ratios
// ============================================================

// --- Direct vs Cached ---

func BenchmarkDirectQuery(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		GetDirect("user-1")
	}
}

func BenchmarkCachedQuery100PercentHit(b *testing.B) {
	cache := NewTTLCache(1 * time.Minute)
	cache.Set("user-1", "cached-value")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		GetWithCache(cache, "user-1")
	}
}

func BenchmarkCachedQuery50PercentHit(b *testing.B) {
	cache := NewTTLCache(1 * time.Minute)
	// Pre-populate half the keys
	for i := 0; i < 50; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100) // 50% hit, 50% miss
		GetWithCache(cache, key)
	}
}

// --- Cache Implementation Comparison ---

func BenchmarkMutexCacheGet(b *testing.B) {
	cache := NewTTLCache(1 * time.Minute)
	cache.Set("key", "value")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkSyncMapGet(b *testing.B) {
	var m sync.Map
	m.Store("key", "value")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Load("key")
	}
}

// --- Concurrent Access ---

func BenchmarkMutexCacheConcurrentRead(b *testing.B) {
	cache := NewTTLCache(1 * time.Minute)
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(fmt.Sprintf("key-%d", i%100))
			i++
		}
	})
}

func BenchmarkSyncMapConcurrentRead(b *testing.B) {
	var m sync.Map
	for i := 0; i < 100; i++ {
		m.Store(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Load(fmt.Sprintf("key-%d", i%100))
			i++
		}
	})
}

// --- Write-Heavy vs Read-Heavy ---

func BenchmarkMutexCacheWriteHeavy(b *testing.B) {
	cache := NewTTLCache(1 * time.Minute)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				cache.Set(key, "value")
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

func BenchmarkSyncMapWriteHeavy(b *testing.B) {
	var m sync.Map

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				m.Store(key, "value")
			} else {
				m.Load(key)
			}
			i++
		}
	})
}
