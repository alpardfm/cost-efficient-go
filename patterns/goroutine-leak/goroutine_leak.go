package goroutine_leak

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ============================================================
// PATTERN 12: Goroutine Leak Detection & Prevention
// ============================================================
// Problem: Goroutines without exit paths accumulate over time:
// - Each goroutine uses ~2-8KB stack memory
// - At 1 leak/second: 86,400 leaked goroutines in 24 hours
// - Memory growth: 172 MB - 691 MB per day (unrecoverable)
// - Service eventually OOM-killed, causing downtime
//
// This pattern demonstrates:
// 1. How goroutines leak (blocked channel, no cancellation)
// 2. Prevention with context.WithCancel + done channels
// 3. Detection by comparing runtime.NumGoroutine()
// 4. Graceful termination with timeout fallback
// ============================================================

// --- Leaky Implementation (Bad) ---

// LeakyServer spawns a goroutine that blocks forever on a channel
// with no exit path. The goroutine will never be garbage collected.
func LeakyServer(requests int) {
	for i := 0; i < requests; i++ {
		ch := make(chan struct{})
		go func(id int) {
			// This goroutine blocks forever — no one closes ch,
			// no select with context, no timeout.
			<-ch
			// Unreachable code
			fmt.Printf("request %d completed\n", id)
		}(i)
		// Simulate: we "forget" about ch and move on
		// The goroutine is now leaked permanently
	}
}

// --- Safe Implementation (Good) ---

// SafeServer spawns goroutines with proper exit paths using
// context cancellation and done channels.
func SafeServer(ctx context.Context, requests int) {
	var wg sync.WaitGroup

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			done := make(chan struct{})

			// Simulate async work
			go func() {
				time.Sleep(1 * time.Millisecond)
				close(done)
			}()

			// Proper exit: either work completes or context is cancelled
			select {
			case <-done:
				// Work completed normally
			case <-ctx.Done():
				// Parent cancelled — clean exit
			}
		}(i)
	}

	wg.Wait()
}

// --- Leak Detector ---

// LeakDetector tracks goroutine count before and after operations
// to identify goroutine leaks.
type LeakDetector struct {
	beforeCount int
	afterCount  int
	label       string
}

// NewLeakDetector creates a detector and snapshots the current goroutine count.
func NewLeakDetector(label string) *LeakDetector {
	// Allow runtime goroutines to settle
	runtime.Gosched()
	return &LeakDetector{
		beforeCount: runtime.NumGoroutine(),
		label:       label,
	}
}

// Snapshot captures the current goroutine count as the "after" state.
func (ld *LeakDetector) Snapshot() {
	runtime.Gosched()
	ld.afterCount = runtime.NumGoroutine()
}

// Leaked returns the number of goroutines that were created but not cleaned up.
func (ld *LeakDetector) Leaked() int {
	delta := ld.afterCount - ld.beforeCount
	if delta < 0 {
		return 0
	}
	return delta
}

// Report prints the leak detection results.
func (ld *LeakDetector) Report() {
	fmt.Printf("  [%s] Before: %d goroutines\n", ld.label, ld.beforeCount)
	fmt.Printf("  [%s] After:  %d goroutines\n", ld.label, ld.afterCount)
	fmt.Printf("  [%s] Leaked: %d goroutines\n", ld.label, ld.Leaked())
}

// --- Graceful Shutdown ---

// GracefulShutdown attempts to stop a long-running goroutine gracefully
// within the given timeout. If the goroutine doesn't stop in time,
// it forces termination via context cancellation.
//
// Returns true if shutdown was graceful, false if forced.
func GracefulShutdown(timeout time.Duration, worker func(ctx context.Context)) bool {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	// Start the worker
	go func() {
		worker(ctx)
		close(done)
	}()

	// Signal graceful stop
	cancel()

	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		return true // Graceful shutdown succeeded
	case <-time.After(timeout):
		// Worker didn't respond to cancellation within timeout.
		// Context is already cancelled — the goroutine will exit
		// on its next ctx.Done() check.
		// Wait a bit more for cleanup.
		select {
		case <-done:
			return false // Terminated after timeout (force path)
		case <-time.After(100 * time.Millisecond):
			return false // Force kill — goroutine abandoned
		}
	}
}

// LongRunningWorker simulates a worker that respects context cancellation.
func LongRunningWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Clean up and exit
			return
		case <-ticker.C:
			// Simulate periodic work
			runtime.Gosched()
		}
	}
}

// StubbornWorker simulates a worker that is slow to respond to cancellation.
func StubbornWorker(ctx context.Context) {
	// Simulates a worker doing blocking I/O that checks context periodically
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(50 * time.Millisecond) // Simulate blocking work
		}
	}
}

// --- Cost Projection ---

// CostProjection24h calculates the cost impact of goroutine leaks over 24 hours.
func CostProjection24h() {
	const (
		leakRatePerSecond = 1
		secondsPerDay     = 86400
		minStackBytes     = 2048 // 2 KB minimum goroutine stack
		maxStackBytes     = 8192 // 8 KB typical goroutine stack
		awsCostPerGBMonth = 3.75 // t3.medium pricing
		instanceRAMGB     = 8    // t3.medium RAM
	)

	totalLeaks := leakRatePerSecond * secondsPerDay // 86,400 goroutines

	minMemoryMB := float64(totalLeaks*minStackBytes) / (1024 * 1024)
	maxMemoryMB := float64(totalLeaks*maxStackBytes) / (1024 * 1024)

	minMemoryGB := minMemoryMB / 1024
	maxMemoryGB := maxMemoryMB / 1024

	// How many instances needed just for leaked goroutines
	minInstances := minMemoryGB / float64(instanceRAMGB)
	maxInstances := maxMemoryGB / float64(instanceRAMGB)

	// Monthly cost of wasted memory
	minMonthlyCost := minMemoryGB * awsCostPerGBMonth
	maxMonthlyCost := maxMemoryGB * awsCostPerGBMonth

	fmt.Println("=== Cost Projection: Goroutine Leak (24 hours) ===")
	fmt.Println()
	fmt.Printf("  Leak rate: %d goroutine/second\n", leakRatePerSecond)
	fmt.Printf("  Duration:  24 hours (%d seconds)\n", secondsPerDay)
	fmt.Printf("  Total leaked goroutines: %d\n", totalLeaks)
	fmt.Println()
	fmt.Printf("  Memory consumed (min ~2KB stack): %.0f MB (%.2f GB)\n", minMemoryMB, minMemoryGB)
	fmt.Printf("  Memory consumed (max ~8KB stack): %.0f MB (%.2f GB)\n", maxMemoryMB, maxMemoryGB)
	fmt.Println()
	fmt.Printf("  Equivalent t3.medium instances (8GB RAM):\n")
	fmt.Printf("    Min: %.1f instances\n", minInstances)
	fmt.Printf("    Max: %.1f instances\n", maxInstances)
	fmt.Println()
	fmt.Printf("  Wasted AWS cost (memory alone):\n")
	fmt.Printf("    Min: $%.2f/month\n", minMonthlyCost)
	fmt.Printf("    Max: $%.2f/month\n", maxMonthlyCost)
	fmt.Println()
	fmt.Println("  Note: Real cost is higher due to GC pressure, CPU overhead,")
	fmt.Println("  and eventual OOM kills requiring service restarts.")

	// Suppress unused variable warnings
	_ = minInstances
	_ = maxInstances
}
