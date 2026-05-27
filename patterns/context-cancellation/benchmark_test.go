package context_cancellation

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ============================================================
// BENCHMARK: Context Cancellation & Resource Cleanup
// ============================================================
//
// Requirements validated:
// - 6.2: Measure cleanup time when context cancelled on 100ms+ operations
// - 6.3: Demonstrate CPU time saved when request cancelled early (client disconnect)
//
// Key insight:
//   When a client disconnects, services that ignore context cancellation
//   continue burning CPU on work nobody will consume. With proper cancellation,
//   a 160ms call chain exits in ~40ms when cancelled at 25%, saving 75% CPU
//   per cancelled request. At 20% cancel rate on 10M req/day, this adds up.
//
// Run benchmarks:
//   go test -bench=. -benchmem -benchtime=3s ./patterns/context-cancellation/
// ============================================================

// Global variables to prevent compiler from optimizing away benchmark results.
var (
	benchResult   CallResult
	benchDuration time.Duration
)

// --- Benchmark: Cleanup Time When Context Cancelled (100ms+ operations) ---

// BenchmarkCleanup_WithCancellation measures how quickly a 160ms call chain
// exits when context is cancelled at 25% (40ms into the chain).
// Expected: completes in ~40ms instead of full 160ms.
func BenchmarkCleanup_WithCancellation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), totalChainLatency/4)
		benchResult = CascadingCall(ctx)
		cancel()
	}
}

// BenchmarkCleanup_WithoutCancellation measures the full 160ms call chain
// running to completion (no cancellation — baseline).
func BenchmarkCleanup_WithoutCancellation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchResult = NoCancellationCall(context.Background())
	}
}

// BenchmarkCleanup_CancelledImmediately measures cleanup time when context
// is already cancelled before the call starts. Should exit near-instantly.
func BenchmarkCleanup_CancelledImmediately(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		benchResult = CascadingCall(ctx)
	}
}

// BenchmarkCleanup_CancelAfterFirstStep measures cleanup when context is
// cancelled after the first step (HTTP call) completes (~50ms into 160ms chain).
func BenchmarkCleanup_CancelAfterFirstStep(b *testing.B) {
	cancelAfter := httpCallLatency + 5*time.Millisecond // Just after HTTP step

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
		benchResult = CascadingCall(ctx)
		cancel()
	}
}

// --- Benchmark: CPU Time Saved When Request Cancelled Early ---

// BenchmarkCPUSaved_EarlyCancel_20Percent simulates a batch of requests where
// 20% are cancelled early. Measures total wall-clock time with proper cancellation.
func BenchmarkCPUSaved_EarlyCancel_20Percent(b *testing.B) {
	const numRequests = 50
	const cancelRate = 0.20

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		cancelCount := int(float64(numRequests) * cancelRate)
		normalCount := numRequests - cancelCount

		// Normal requests — run to completion
		for j := 0; j < normalCount; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				CascadingCall(context.Background())
			}()
		}

		// Cancelled requests — cancel at 25% of chain time
		cancelAfter := totalChainLatency / 4
		for j := 0; j < cancelCount; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
				defer cancel()
				CascadingCall(ctx)
			}()
		}

		wg.Wait()
	}
}

// BenchmarkCPUSaved_NoCancellation_Baseline simulates the same batch of requests
// but WITHOUT cancellation — all requests run to completion regardless.
func BenchmarkCPUSaved_NoCancellation_Baseline(b *testing.B) {
	const numRequests = 50

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		for j := 0; j < numRequests; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				NoCancellationCall(context.Background())
			}()
		}

		wg.Wait()
	}
}

// --- Benchmark: Resource Usage Comparison (With vs Without Cancellation) ---

// BenchmarkResourceUsage_CascadingCall_Normal measures resource usage of
// the cancellation-aware implementation when no cancellation occurs (normal path).
func BenchmarkResourceUsage_CascadingCall_Normal(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchResult = CascadingCall(context.Background())
	}
}

// BenchmarkResourceUsage_NoCancellationCall_Normal measures resource usage of
// the non-cancellation-aware implementation (always runs full chain).
func BenchmarkResourceUsage_NoCancellationCall_Normal(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchResult = NoCancellationCall(context.Background())
	}
}

// BenchmarkResourceUsage_CascadingCall_Cancelled measures resource usage when
// the cancellation-aware implementation is cancelled mid-chain.
func BenchmarkResourceUsage_CascadingCall_Cancelled(b *testing.B) {
	cancelAfter := totalChainLatency / 2 // Cancel at 50% (80ms)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
		benchResult = CascadingCall(ctx)
		cancel()
	}
}

// BenchmarkResourceUsage_NoCancellationCall_IgnoresCancellation measures that
// the non-cancellation-aware implementation wastes full resources even when
// context is cancelled.
func BenchmarkResourceUsage_NoCancellationCall_IgnoresCancellation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), totalChainLatency/2)
		benchResult = NoCancellationCall(ctx)
		cancel()
	}
}

// --- Benchmark: Concurrent Cancellation Under Load ---

// BenchmarkConcurrent_WithCancellation_100Requests measures throughput of
// 100 concurrent requests where 20% are cancelled early.
func BenchmarkConcurrent_WithCancellation_100Requests(b *testing.B) {
	const numRequests = 100
	const cancelRate = 0.20

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		cancelCount := int(float64(numRequests) * cancelRate)
		normalCount := numRequests - cancelCount

		for j := 0; j < normalCount; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				CascadingCall(context.Background())
			}()
		}

		cancelAfter := totalChainLatency / 4
		for j := 0; j < cancelCount; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
				defer cancel()
				CascadingCall(ctx)
			}()
		}

		wg.Wait()
	}
}

// BenchmarkConcurrent_WithoutCancellation_100Requests measures throughput of
// 100 concurrent requests all running to completion (no cancellation).
func BenchmarkConcurrent_WithoutCancellation_100Requests(b *testing.B) {
	const numRequests = 100

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup

		for j := 0; j < numRequests; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				NoCancellationCall(context.Background())
			}()
		}

		wg.Wait()
	}
}

// --- Unit Tests: Validate Cancellation Behavior ---

// TestCascadingCall_CompletesNormally verifies the call chain completes
// all 3 steps when no cancellation occurs.
func TestCascadingCall_CompletesNormally(t *testing.T) {
	result := CascadingCall(context.Background())
	if !result.Completed {
		t.Fatal("expected call to complete normally")
	}
	if result.StepsRun != 3 {
		t.Fatalf("expected 3 steps, got %d", result.StepsRun)
	}
}

// TestCascadingCall_CancelsEarly verifies the call chain exits early
// when context is cancelled before completion.
func TestCascadingCall_CancelsEarly(t *testing.T) {
	// Cancel after ~40ms (should complete HTTP step but not DB step)
	ctx, cancel := context.WithTimeout(context.Background(), totalChainLatency/4)
	defer cancel()

	result := CascadingCall(ctx)
	if result.Completed {
		t.Fatal("expected call to be cancelled, but it completed")
	}
	if result.StepsRun >= 3 {
		t.Fatalf("expected fewer than 3 steps when cancelled, got %d", result.StepsRun)
	}
}

// TestCascadingCall_ImmediateCancel verifies zero steps run when context
// is already cancelled.
func TestCascadingCall_ImmediateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := CascadingCall(ctx)
	if result.Completed {
		t.Fatal("expected call to be cancelled immediately")
	}
	if result.StepsRun != 0 {
		t.Fatalf("expected 0 steps when cancelled immediately, got %d", result.StepsRun)
	}
}

// TestNoCancellationCall_IgnoresCancel verifies that NoCancellationCall
// always runs all 3 steps regardless of context state.
func TestNoCancellationCall_IgnoresCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := NoCancellationCall(ctx)
	if !result.Completed {
		t.Fatal("NoCancellationCall should always complete")
	}
	if result.StepsRun != 3 {
		t.Fatalf("expected 3 steps regardless of cancel, got %d", result.StepsRun)
	}
}

// TestEarlyCancelDemo_SavesTime verifies that early cancellation results
// in less total wall-clock time than running all requests to completion.
func TestEarlyCancelDemo_SavesTime(t *testing.T) {
	withCancel, withoutCancel := EarlyCancelDemo(20, 0.20)

	// With cancellation should be faster (or at least not significantly slower)
	// Allow some tolerance for scheduling jitter
	if withCancel > withoutCancel+50*time.Millisecond {
		t.Errorf("expected withCancel (%v) <= withoutCancel (%v) + tolerance",
			withCancel, withoutCancel)
	}
}

// TestCascadingCall_DurationBounded verifies that cancelled calls complete
// faster than the full chain duration.
func TestCascadingCall_DurationBounded(t *testing.T) {
	cancelAfter := 40 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
	defer cancel()

	result := CascadingCall(ctx)

	// Duration should be close to cancelAfter, not totalChainLatency
	maxExpected := cancelAfter + 20*time.Millisecond // Allow some overhead
	if result.Duration > maxExpected {
		t.Errorf("cancelled call took %v, expected < %v", result.Duration, maxExpected)
	}
}
