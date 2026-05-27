package profiling_benchmarking

import (
	"runtime"
	"sort"
	"time"
)

// ============================================================
// PATTERN 5: Profiling & Benchmarking Techniques
// ============================================================
// This pattern demonstrates how to measure performance
// correctly in Go and avoid common benchmarking pitfalls.
//
// Topics:
// 1. runtime.MemStats for memory measurement
// 2. Avoiding compiler optimization elimination
// 3. Warm-up vs cold-start measurement
// 4. Percentile-based timing (P50, P95, P99)
// ============================================================

// --- Workloads to measure ---

// SortInts sorts a copy of the input slice (simulates CPU-bound work)
func SortInts(data []int) []int {
	cp := make([]int, len(data))
	copy(cp, data)
	sort.Ints(cp)
	return cp
}

// AllocateSlices creates n slices of given size (simulates allocation-heavy work)
func AllocateSlices(n, size int) [][]byte {
	result := make([][]byte, n)
	for i := range result {
		result[i] = make([]byte, size)
	}
	return result
}

// ProcessBatch simulates a batch processing workload
func ProcessBatch(items []int) int {
	total := 0
	for _, item := range items {
		// Simulate some computation
		total += item * item
	}
	return total
}

// --- Measurement utilities ---

// MemUsage returns current heap allocation in bytes
func MemUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// MeasureAllocs measures heap allocations of a function
func MeasureAllocs(fn func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// MeasureTime measures execution time with percentiles
func MeasureTime(fn func(), iterations int) (p50, p95, p99 time.Duration) {
	durations := make([]time.Duration, iterations)

	// Warm-up (discard first 10% of results)
	warmup := iterations / 10
	for i := 0; i < warmup; i++ {
		fn()
	}

	// Actual measurement
	for i := 0; i < iterations; i++ {
		start := time.Now()
		fn()
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p50 = durations[iterations*50/100]
	p95 = durations[iterations*95/100]
	p99 = durations[iterations*99/100]
	return
}
