// Package main demonstrates profiling and benchmarking techniques.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/profiling-benchmarking/examples/
package main

import (
	"fmt"
	"math/rand"
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

// --- Sink to prevent compiler elimination ---
var sink interface{}

func main() {
	fmt.Println("=== Profiling & Benchmarking Techniques ===")
	fmt.Println()

	// 1. Memory measurement
	fmt.Println("--- Memory Measurement ---")
	allocs := MeasureAllocs(func() {
		sink = AllocateSlices(1000, 1024)
	})
	fmt.Printf("AllocateSlices(1000, 1KB): %d bytes allocated (%.2f MB)\n", allocs, float64(allocs)/1024/1024)

	allocs = MeasureAllocs(func() {
		sink = AllocateSlices(1000, 4096)
	})
	fmt.Printf("AllocateSlices(1000, 4KB): %d bytes allocated (%.2f MB)\n", allocs, float64(allocs)/1024/1024)
	fmt.Println()

	// 2. Percentile-based timing
	fmt.Println("--- Percentile Timing (Sort 10K ints) ---")
	data := make([]int, 10000)
	for i := range data {
		data[i] = rand.Intn(100000)
	}

	p50, p95, p99 := MeasureTime(func() {
		sink = SortInts(data)
	}, 1000)
	fmt.Printf("P50: %v\n", p50)
	fmt.Printf("P95: %v\n", p95)
	fmt.Printf("P99: %v\n", p99)
	fmt.Printf("P99/P50 ratio: %.1fx (tail latency amplification)\n", float64(p99)/float64(p50))
	fmt.Println()

	// 3. Batch processing measurement
	fmt.Println("--- Batch Processing ---")
	batch := make([]int, 100000)
	for i := range batch {
		batch[i] = rand.Intn(1000)
	}

	p50, p95, p99 = MeasureTime(func() {
		sink = ProcessBatch(batch)
	}, 1000)
	fmt.Printf("ProcessBatch(100K items):\n")
	fmt.Printf("  P50: %v\n", p50)
	fmt.Printf("  P95: %v\n", p95)
	fmt.Printf("  P99: %v\n", p99)
	fmt.Println()

	// 4. GC impact demonstration
	fmt.Println("--- GC Impact ---")
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Printf("GC cycles so far: %d\n", stats.NumGC)
	fmt.Printf("Total GC pause: %v\n", time.Duration(stats.PauseTotalNs))
	fmt.Printf("Last GC pause: %v\n", time.Duration(stats.PauseNs[(stats.NumGC+255)%256]))

	_ = sink // prevent unused warning
}
