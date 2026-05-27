package context_cancellation

import (
	"context"
	"sync"
	"time"
)

// ============================================================
// PATTERN: Context Cancellation & Resource Cleanup
// ============================================================
// Problem: When a client disconnects or a timeout fires, services
// that ignore context cancellation continue burning CPU/memory on
// work that nobody will ever consume.
//
// This module provides the core functions demonstrating:
// 1. Cascading cancellation: HTTP → DB → Cache — cancel propagates
// 2. No cancellation: request runs to completion despite disconnect
// 3. Anti-pattern: context.Background() in goroutine loses parent cancel
// 4. Early cancel: 20% requests cancelled early saves significant CPU
// ============================================================

// --- Simulated Service Latencies ---

const (
	httpCallLatency   = 50 * time.Millisecond                                // Simulated HTTP downstream call
	dbQueryLatency    = 80 * time.Millisecond                                // Simulated DB query
	cacheWriteLatency = 30 * time.Millisecond                                // Simulated cache write
	totalChainLatency = httpCallLatency + dbQueryLatency + cacheWriteLatency // 160ms total
)

// --- Result Types ---

// CallResult holds the outcome of a service call chain.
type CallResult struct {
	Completed   bool
	Duration    time.Duration
	StepsRun    int
	CancelledAt string
}

// --- Cascading Cancellation Pattern (GOOD) ---

// CascadingCall simulates an HTTP → DB → Cache call chain with proper
// context cancellation propagation. If the parent context is cancelled,
// each step checks before proceeding, saving CPU on abandoned work.
func CascadingCall(ctx context.Context) CallResult {
	start := time.Now()
	steps := 0

	// Step 1: HTTP downstream call
	if err := simulateWork(ctx, httpCallLatency); err != nil {
		return CallResult{
			Completed:   false,
			Duration:    time.Since(start),
			StepsRun:    steps,
			CancelledAt: "http_call",
		}
	}
	steps++

	// Step 2: DB query
	if err := simulateWork(ctx, dbQueryLatency); err != nil {
		return CallResult{
			Completed:   false,
			Duration:    time.Since(start),
			StepsRun:    steps,
			CancelledAt: "db_query",
		}
	}
	steps++

	// Step 3: Cache write
	if err := simulateWork(ctx, cacheWriteLatency); err != nil {
		return CallResult{
			Completed:   false,
			Duration:    time.Since(start),
			StepsRun:    steps,
			CancelledAt: "cache_write",
		}
	}
	steps++

	return CallResult{
		Completed: true,
		Duration:  time.Since(start),
		StepsRun:  steps,
	}
}

// --- No Cancellation Pattern (BAD — wastes resources) ---

// NoCancellationCall simulates the same call chain but ignores context
// cancellation entirely. Even if the client disconnects, all three steps
// run to completion, wasting CPU and I/O.
func NoCancellationCall(ctx context.Context) CallResult {
	start := time.Now()
	steps := 0

	// Step 1: HTTP downstream call — ignores cancellation
	time.Sleep(httpCallLatency)
	steps++

	// Step 2: DB query — ignores cancellation
	time.Sleep(dbQueryLatency)
	steps++

	// Step 3: Cache write — ignores cancellation
	time.Sleep(cacheWriteLatency)
	steps++

	return CallResult{
		Completed: true,
		Duration:  time.Since(start),
		StepsRun:  steps,
	}
}

// --- Anti-Pattern: context.Background() in Goroutine ---

// AntiPatternDemo demonstrates the common mistake of using context.Background()
// inside a goroutine that should inherit the parent context. The goroutine
// becomes "orphaned" — it cannot be cancelled when the parent is done.
func AntiPatternDemo(parentCtx context.Context) (goodResult, badResult CallResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// GOOD: goroutine uses parent context — cancellable
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := CascadingCall(parentCtx)
		mu.Lock()
		goodResult = result
		mu.Unlock()
	}()

	// BAD: goroutine uses context.Background() — NOT cancellable
	// Even if parentCtx is cancelled, this goroutine runs to completion.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Anti-pattern: context.Background() ignores parent cancellation
		result := CascadingCall(context.Background())
		mu.Lock()
		badResult = result
		mu.Unlock()
	}()

	wg.Wait()
	return goodResult, badResult
}

// --- Early Cancel Simulation ---

// EarlyCancelDemo simulates a batch of requests where some are cancelled early
// (client disconnect). It measures CPU time saved by proper cancellation vs
// running all requests to completion.
func EarlyCancelDemo(totalRequests int, cancelRate float64) (withCancel, withoutCancel time.Duration) {
	cancelCount := int(float64(totalRequests) * cancelRate)
	normalCount := totalRequests - cancelCount

	// --- With proper cancellation ---
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < normalCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			CascadingCall(ctx)
		}()
	}

	// Cancelled requests: cancel after ~25% of total chain time
	cancelAfter := totalChainLatency / 4 // Cancel after ~40ms
	for i := 0; i < cancelCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
			defer cancel()
			CascadingCall(ctx)
		}()
	}

	wg.Wait()
	withCancel = time.Since(start)

	// --- Without cancellation (all run to completion) ---
	start = time.Now()
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			NoCancellationCall(context.Background())
		}()
	}
	wg.Wait()
	withoutCancel = time.Since(start)

	return withCancel, withoutCancel
}

// --- Helper: Simulate Work with Context Check ---

// simulateWork simulates a blocking operation that respects context cancellation.
// If the context is cancelled before the work completes, it returns immediately.
func simulateWork(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}
