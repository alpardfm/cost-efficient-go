package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// ============================================================
// Benchmarks: sync.Pool Buffer Reuse vs Naive Allocation
// ============================================================
// Requirements 1.2: Measure allocation difference between new() vs pool reuse
// Requirements 1.3: Demonstrate GC pause time reduction at 10K+ ops/sec
// ============================================================

// Global vars to prevent compiler optimization
var (
	globalBuf []byte
)

// --- Benchmark: Naive vs Pooled at Various Buffer Sizes ---

func BenchmarkNaiveAlloc_1KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = NaiveBufferAlloc(1024)
	}
}

func BenchmarkPooledAlloc_1KB(b *testing.B) {
	pool := NewBufferPool(1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = PooledBufferAlloc(pool)
	}
}

func BenchmarkNaiveAlloc_4KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = NaiveBufferAlloc(4096)
	}
}

func BenchmarkPooledAlloc_4KB(b *testing.B) {
	pool := NewBufferPool(4096)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = PooledBufferAlloc(pool)
	}
}

func BenchmarkNaiveAlloc_64KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = NaiveBufferAlloc(65536)
	}
}

func BenchmarkPooledAlloc_64KB(b *testing.B) {
	pool := NewBufferPool(65536)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalBuf = PooledBufferAlloc(pool)
	}
}

// --- Benchmark: GC Pause Time at High Throughput (10K+ ops/sec) ---
// Measures GC pause time difference between naive allocation (creates garbage)
// and pooled allocation (reuses buffers, less GC pressure).

func BenchmarkGCPause_Naive_10K(b *testing.B) {
	const opsPerMeasurement = 10_000
	const bufSize = 4096

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Force a clean GC state
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// Perform 10K allocations (naive — creates garbage)
		for j := 0; j < opsPerMeasurement; j++ {
			globalBuf = NaiveBufferAlloc(bufSize)
		}

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		// Record GC pause total (prevents compiler from optimizing away ReadMemStats)
		_ = memAfter.PauseTotalNs - memBefore.PauseTotalNs
	}
}

func BenchmarkGCPause_Pooled_10K(b *testing.B) {
	const opsPerMeasurement = 10_000
	const bufSize = 4096
	pool := NewBufferPool(bufSize)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Force a clean GC state
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// Perform 10K allocations (pooled — reuses buffers)
		for j := 0; j < opsPerMeasurement; j++ {
			globalBuf = PooledBufferAlloc(pool)
		}

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		// Record GC pause total (prevents compiler from optimizing away ReadMemStats)
		_ = memAfter.PauseTotalNs - memBefore.PauseTotalNs
	}
}

// BenchmarkGCPauseDelta measures the actual GC pause time difference
// between naive and pooled approaches at sustained high throughput.
func BenchmarkGCPauseDelta(b *testing.B) {
	const bufSize = 4096
	const opsPerRound = 50_000 // Sustained high throughput

	b.Run("naive", func(b *testing.B) {
		b.ReportAllocs()
		runtime.GC()
		var memStart runtime.MemStats
		runtime.ReadMemStats(&memStart)

		for i := 0; i < b.N; i++ {
			for j := 0; j < opsPerRound; j++ {
				globalBuf = NaiveBufferAlloc(bufSize)
			}
		}

		var memEnd runtime.MemStats
		runtime.ReadMemStats(&memEnd)
		b.ReportMetric(float64(memEnd.PauseTotalNs-memStart.PauseTotalNs)/float64(b.N), "gc-pause-ns/op")
	})

	b.Run("pooled", func(b *testing.B) {
		pool := NewBufferPool(bufSize)
		b.ReportAllocs()
		runtime.GC()
		var memStart runtime.MemStats
		runtime.ReadMemStats(&memStart)

		for i := 0; i < b.N; i++ {
			for j := 0; j < opsPerRound; j++ {
				globalBuf = PooledBufferAlloc(pool)
			}
		}

		var memEnd runtime.MemStats
		runtime.ReadMemStats(&memEnd)
		b.ReportMetric(float64(memEnd.PauseTotalNs-memStart.PauseTotalNs)/float64(b.N), "gc-pause-ns/op")
	})
}

// --- Benchmark: Concurrent Access with GOMAXPROCS Workers ---
// Validates pool behavior under contention from multiple goroutines.

func BenchmarkConcurrentNaive(b *testing.B) {
	const bufSize = 4096
	b.ReportAllocs()
	b.SetParallelism(runtime.GOMAXPROCS(0))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			globalBuf = NaiveBufferAlloc(bufSize)
		}
	})
}

func BenchmarkConcurrentPooled(b *testing.B) {
	const bufSize = 4096
	pool := NewBufferPool(bufSize)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetParallelism(runtime.GOMAXPROCS(0))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			globalBuf = PooledBufferAlloc(pool)
		}
	})
}

// BenchmarkConcurrentPooled_VaryingWorkers benchmarks pool performance
// at different concurrency levels to show scaling behavior.
func BenchmarkConcurrentPooled_VaryingWorkers(b *testing.B) {
	const bufSize = 4096
	workers := uniqueWorkers(1, 2, 4, 8, runtime.GOMAXPROCS(0))

	for _, numWorkers := range workers {
		b.Run(workerLabel(numWorkers), func(b *testing.B) {
			pool := NewBufferPool(bufSize)
			b.ResetTimer()
			b.ReportAllocs()
			b.SetParallelism(numWorkers)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					globalBuf = PooledBufferAlloc(pool)
				}
			})
		})
	}
}

// BenchmarkConcurrentNaive_VaryingWorkers benchmarks naive allocation
// at different concurrency levels for comparison.
func BenchmarkConcurrentNaive_VaryingWorkers(b *testing.B) {
	const bufSize = 4096
	workers := uniqueWorkers(1, 2, 4, 8, runtime.GOMAXPROCS(0))

	for _, numWorkers := range workers {
		b.Run(workerLabel(numWorkers), func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(numWorkers)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					globalBuf = NaiveBufferAlloc(bufSize)
				}
			})
		})
	}
}

// --- Benchmark: Small Object (< 64 bytes) — Pool Overhead Exceeds Benefit ---

var globalSmallObj *SmallObject

func BenchmarkSmallObject_Naive(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		globalSmallObj = &SmallObject{ID: int32(i), Value: int32(i * 2)}
	}
}

func BenchmarkSmallObject_Pooled(b *testing.B) {
	smallPool := sync.Pool{
		New: func() interface{} {
			return &SmallObject{}
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		obj := smallPool.Get().(*SmallObject)
		obj.ID = int32(i)
		obj.Value = int32(i * 2)
		globalSmallObj = obj
		smallPool.Put(obj)
	}
}

// --- Helper ---

func workerLabel(n int) string {
	return fmt.Sprintf("workers=%d", n)
}

// uniqueWorkers deduplicates worker counts (e.g., when GOMAXPROCS == 8).
func uniqueWorkers(counts ...int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(counts))
	for _, c := range counts {
		if !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}
	return result
}
