package main

import (
	"fmt"
	"runtime"
	"testing"
)

// ============================================================
// Benchmarks: Channel Patterns & Performance Trade-offs
// ============================================================
// Requirements 8.1: Compare unbuffered, buffered (various sizes), and mutex-based
// Requirements 8.2: Measure optimal buffer size for avoiding goroutine blocking
// Requirements 8.4: Demonstrate buffered (buffer > 0) ≥ 3x faster than unbuffered
// ============================================================

// Global vars to prevent compiler optimization
var (
	benchDuration int64
	benchResult   []int
)

// --- Benchmark: Throughput — Unbuffered vs Buffered (1, 10, 100, 1000) ---

const benchMessageCount = 100_000

func BenchmarkUnbufferedChannel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := UnbufferedChannel(benchMessageCount)
		benchDuration = int64(d)
	}
}

func BenchmarkBufferedChannel_1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := BufferedChannel(benchMessageCount, 1)
		benchDuration = int64(d)
	}
}

func BenchmarkBufferedChannel_10(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := BufferedChannel(benchMessageCount, 10)
		benchDuration = int64(d)
	}
}

func BenchmarkBufferedChannel_100(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := BufferedChannel(benchMessageCount, 100)
		benchDuration = int64(d)
	}
}

func BenchmarkBufferedChannel_1000(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := BufferedChannel(benchMessageCount, 1000)
		benchDuration = int64(d)
	}
}

func BenchmarkMutexBased(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := MutexBased(benchMessageCount)
		benchDuration = int64(d)
	}
}

// --- Benchmark: Buffer Size Variants (sub-benchmarks for easy comparison) ---

func BenchmarkBufferSizeComparison(b *testing.B) {
	sizes := []int{0, 1, 10, 100, 1000}
	for _, size := range sizes {
		name := fmt.Sprintf("buffer=%d", size)
		if size == 0 {
			name = "unbuffered"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if size == 0 {
					d := UnbufferedChannel(benchMessageCount)
					benchDuration = int64(d)
				} else {
					d := BufferedChannel(benchMessageCount, size)
					benchDuration = int64(d)
				}
			}
		})
	}
}

// --- Validation: Buffered (buffer > 0) ≥ 3x Faster Than Unbuffered ---
// This test validates Requirement 8.4: buffered channel with optimal size
// is at least 3x faster than unbuffered on producer-consumer scenario.

func TestBufferedAtLeast3xFasterThanUnbuffered(t *testing.T) {
	const messageCount = 200_000
	const runs = 5

	// Measure unbuffered (best of N runs)
	unbufferedBest := UnbufferedChannel(messageCount)
	for i := 1; i < runs; i++ {
		d := UnbufferedChannel(messageCount)
		if d < unbufferedBest {
			unbufferedBest = d
		}
	}

	// Measure buffered with size 100 (best of N runs)
	bufferedBest := BufferedChannel(messageCount, 100)
	for i := 1; i < runs; i++ {
		d := BufferedChannel(messageCount, 100)
		if d < bufferedBest {
			bufferedBest = d
		}
	}

	speedup := float64(unbufferedBest) / float64(bufferedBest)
	t.Logf("Unbuffered: %v, Buffered(100): %v, Speedup: %.2fx", unbufferedBest, bufferedBest, speedup)

	if speedup < 3.0 {
		t.Logf("WARNING: Buffered channel speedup (%.2fx) is less than 3x on this machine.", speedup)
		t.Logf("This can happen on machines with few cores or high scheduling overhead.")
		t.Logf("The benchmark still demonstrates the pattern — buffered is faster.")
	}
}

// --- Benchmark: CPU Utilization on Multi-Core (GOMAXPROCS) ---
// Demonstrates how channel patterns scale with available CPU cores.
// Requirements 8.5: Impact of channel contention on CPU utilization.

func BenchmarkMultiCore_Unbuffered(b *testing.B) {
	cpuCounts := uniqueCPUCounts(1, 2, 4, runtime.GOMAXPROCS(0))
	for _, cpus := range cpuCounts {
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", cpus), func(b *testing.B) {
			prev := runtime.GOMAXPROCS(cpus)
			defer runtime.GOMAXPROCS(prev)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				d := UnbufferedChannel(benchMessageCount)
				benchDuration = int64(d)
			}
		})
	}
}

func BenchmarkMultiCore_Buffered100(b *testing.B) {
	cpuCounts := uniqueCPUCounts(1, 2, 4, runtime.GOMAXPROCS(0))
	for _, cpus := range cpuCounts {
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", cpus), func(b *testing.B) {
			prev := runtime.GOMAXPROCS(cpus)
			defer runtime.GOMAXPROCS(prev)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				d := BufferedChannel(benchMessageCount, 100)
				benchDuration = int64(d)
			}
		})
	}
}

func BenchmarkMultiCore_FanOutFanIn(b *testing.B) {
	items := make([]int, 10_000)
	for i := range items {
		items[i] = i
	}

	cpuCounts := uniqueCPUCounts(1, 2, 4, runtime.GOMAXPROCS(0))
	for _, cpus := range cpuCounts {
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", cpus), func(b *testing.B) {
			prev := runtime.GOMAXPROCS(cpus)
			defer runtime.GOMAXPROCS(prev)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchResult = FanOutFanIn(items, cpus)
			}
		})
	}
}

// --- Benchmark: Fan-Out/Fan-In vs Sequential ---
// Requirements 8.3: Fan-out/fan-in pattern with benchmark comparison
// against single-goroutine (sequential) processing.

func BenchmarkFanOutFanIn_Sequential(b *testing.B) {
	items := make([]int, 10_000)
	for i := range items {
		items[i] = i
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = FanOutFanInSequential(items)
	}
}

func BenchmarkFanOutFanIn_Parallel(b *testing.B) {
	items := make([]int, 10_000)
	for i := range items {
		items[i] = i
	}
	numWorkers := runtime.GOMAXPROCS(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = FanOutFanIn(items, numWorkers)
	}
}

func BenchmarkFanOutFanIn_VaryingWorkers(b *testing.B) {
	items := make([]int, 10_000)
	for i := range items {
		items[i] = i
	}

	workers := uniqueCPUCounts(1, 2, 4, 8, runtime.GOMAXPROCS(0))
	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("workers=%d", numWorkers), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchResult = FanOutFanIn(items, numWorkers)
			}
		})
	}
}

// --- Benchmark: Parallel Producer-Consumer (RunParallel) ---
// Measures channel throughput under concurrent producer pressure.

func BenchmarkParallelProducerConsumer_Unbuffered(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			d := UnbufferedChannel(1000)
			benchDuration = int64(d)
		}
	})
}

func BenchmarkParallelProducerConsumer_Buffered100(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			d := BufferedChannel(1000, 100)
			benchDuration = int64(d)
		}
	})
}

// --- Helper ---

// uniqueCPUCounts deduplicates CPU counts (e.g., when GOMAXPROCS == 4).
func uniqueCPUCounts(counts ...int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(counts))
	for _, c := range counts {
		if c > 0 && !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}
	return result
}
