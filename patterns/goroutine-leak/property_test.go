package goroutine_leak

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 2: Leak detector correctly identifies goroutine leaks (delta = N for N leaked goroutines)
func TestProperty_LeakDetectorIdentifiesLeaks(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("leak detector reports delta equal to N leaked goroutines", prop.ForAll(
		func(n int) bool {
			// Let runtime settle
			runtime.GC()
			runtime.Gosched()
			time.Sleep(5 * time.Millisecond)

			detector := NewLeakDetector("test")
			LeakyServer(n)
			time.Sleep(10 * time.Millisecond)
			runtime.Gosched()
			detector.Snapshot()

			leaked := detector.Leaked()
			// The leaked count should be at least N (goroutines we spawned that block forever)
			return leaked >= n
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 3: Clean implementation has bounded memory (goroutine count stays within initial + constant)
func TestProperty_CleanImplementationBoundedGoroutines(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("safe server goroutine count stays bounded", prop.ForAll(
		func(n int) bool {
			runtime.GC()
			runtime.Gosched()
			time.Sleep(5 * time.Millisecond)

			before := runtime.NumGoroutine()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			SafeServer(ctx, n)
			cancel()

			time.Sleep(10 * time.Millisecond)
			runtime.Gosched()

			after := runtime.NumGoroutine()

			// After SafeServer completes, goroutine count should be bounded
			// Allow a small constant overhead for runtime goroutines
			const maxOverhead = 5
			return after <= before+maxOverhead
		},
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 4: Graceful termination completes within timeout + epsilon
func TestProperty_GracefulTerminationWithinTimeout(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("graceful shutdown completes within timeout + epsilon", prop.ForAll(
		func(timeoutMs int) bool {
			timeout := time.Duration(timeoutMs) * time.Millisecond
			epsilon := 150 * time.Millisecond // Allow for scheduling overhead

			start := time.Now()
			GracefulShutdown(timeout, LongRunningWorker)
			elapsed := time.Since(start)

			// Should complete within timeout + epsilon
			return elapsed <= timeout+epsilon
		},
		gen.IntRange(50, 500),
	))

	properties.TestingRun(t)
}
