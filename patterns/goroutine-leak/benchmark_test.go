package goroutine_leak

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// Global variables to prevent compiler optimization
var (
	globalGoroutineCount int
	globalMemStats       runtime.MemStats
	globalBool           bool
)

// ============================================================
// Benchmark: Goroutine Growth Rate (Leaky vs Safe)
// Measures runtime.NumGoroutine() growth over 1000 iterations
// Validates: Requirement 2.3
// ============================================================

// BenchmarkLeakyServer_GoroutineGrowth measures goroutine accumulation
// when goroutines have no exit path. Each iteration leaks goroutines
// that are never cleaned up.
func BenchmarkLeakyServer_GoroutineGrowth(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		LeakyServer(1)
		globalGoroutineCount = runtime.NumGoroutine()
	}
}

// BenchmarkSafeServer_GoroutineGrowth measures goroutine count stability
// when goroutines properly exit via context cancellation.
func BenchmarkSafeServer_GoroutineGrowth(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		SafeServer(ctx, 1)
		cancel()
		globalGoroutineCount = runtime.NumGoroutine()
	}
}

// BenchmarkLeakyServer_1000Iterations measures total goroutine growth
// after 1000 leaked goroutines — simulates sustained leak over time.
func BenchmarkLeakyServer_1000Iterations(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		before := runtime.NumGoroutine()
		LeakyServer(1000)
		runtime.Gosched()
		after := runtime.NumGoroutine()
		globalGoroutineCount = after - before
	}
}

// BenchmarkSafeServer_1000Iterations measures goroutine count after
// 1000 properly-managed goroutines — count should remain bounded.
func BenchmarkSafeServer_1000Iterations(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		before := runtime.NumGoroutine()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		SafeServer(ctx, 1000)
		cancel()
		runtime.Gosched()
		after := runtime.NumGoroutine()
		globalGoroutineCount = after - before
	}
}

// ============================================================
// Benchmark: Memory Growth (Leaky vs Clean)
// Measures heap allocation differences between implementations
// Validates: Requirement 2.3
// ============================================================

// BenchmarkLeakyServer_MemoryGrowth measures memory allocated by
// goroutines that never terminate — each leaked goroutine holds
// its stack (~2-8KB) indefinitely.
func BenchmarkLeakyServer_MemoryGrowth(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		LeakyServer(100)
	}
}

// BenchmarkSafeServer_MemoryGrowth measures memory usage when
// goroutines properly terminate — memory is reclaimed by GC.
func BenchmarkSafeServer_MemoryGrowth(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		SafeServer(ctx, 100)
		cancel()
	}
}

// BenchmarkLeakyServer_MemStats captures detailed memory statistics
// after sustained goroutine leaks to show RSS/heap growth.
func BenchmarkLeakyServer_MemStats(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		LeakyServer(100)
		runtime.ReadMemStats(&globalMemStats)
	}
}

// BenchmarkSafeServer_MemStats captures memory statistics after
// properly-cleaned goroutines to contrast with leaky version.
func BenchmarkSafeServer_MemStats(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		SafeServer(ctx, 100)
		cancel()
		runtime.GC()
		runtime.ReadMemStats(&globalMemStats)
	}
}

// ============================================================
// Benchmark: Graceful Shutdown Performance
// Measures overhead of graceful termination mechanism
// ============================================================

// BenchmarkGracefulShutdown_Responsive measures shutdown time for
// a worker that responds immediately to context cancellation.
func BenchmarkGracefulShutdown_Responsive(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		globalBool = GracefulShutdown(500*time.Millisecond, LongRunningWorker)
	}
}

// BenchmarkGracefulShutdown_Stubborn measures shutdown time for
// a worker that is slow to respond to cancellation.
func BenchmarkGracefulShutdown_Stubborn(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		globalBool = GracefulShutdown(100*time.Millisecond, StubbornWorker)
	}
}

// ============================================================
// Benchmark: LeakDetector Overhead
// Measures the cost of leak detection instrumentation
// ============================================================

// BenchmarkLeakDetector_Overhead measures the performance cost
// of using the LeakDetector to monitor goroutine counts.
func BenchmarkLeakDetector_Overhead(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ld := NewLeakDetector("bench")
		ld.Snapshot()
		globalGoroutineCount = ld.Leaked()
	}
}
