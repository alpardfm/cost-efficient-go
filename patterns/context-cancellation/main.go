package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================
// PATTERN 16: Context Cancellation & Resource Cleanup
// ============================================================
// Problem: When a client disconnects or a timeout fires, services
// that ignore context cancellation continue burning CPU/memory on
// work that nobody will ever consume.
//
// This pattern demonstrates:
// 1. Cascading cancellation: HTTP → DB → Cache — cancel propagates
// 2. No cancellation: request runs to completion despite disconnect
// 3. Anti-pattern: context.Background() in goroutine loses parent cancel
// 4. Early cancel: 20% requests cancelled early saves significant CPU
// 5. Impact: 20% early cancellation at 10M req/day saves ~$XXX/month
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

// EarlyCancelDemo simulates a batch of requests where 20% are cancelled early
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

// --- Cost Projection ---

func calculateCostProjection() {
	fmt.Println("=== Cost Projection: Context Cancellation at Scale ===")
	fmt.Println()

	// Parameters
	requestsPerDay := 10_000_000 // 10M requests/day
	cancelRate := 0.20           // 20% cancelled early
	cancelledPerDay := int(float64(requestsPerDay) * cancelRate)
	normalPerDay := requestsPerDay - cancelledPerDay

	// Timing assumptions (from simulated latencies)
	fullChainMs := float64(totalChainLatency.Milliseconds()) // 160ms
	cancelledAtMs := fullChainMs * 0.25                      // Cancel at 25% = 40ms

	// CPU time calculations
	cpuTimeNormalPerDay := float64(normalPerDay) * fullChainMs              // ms
	cpuTimeCancelledWithCancel := float64(cancelledPerDay) * cancelledAtMs  // ms (early exit)
	cpuTimeCancelledWithoutCancel := float64(cancelledPerDay) * fullChainMs // ms (runs to completion)

	totalCPUWithCancel := cpuTimeNormalPerDay + cpuTimeCancelledWithCancel
	totalCPUWithoutCancel := cpuTimeNormalPerDay + cpuTimeCancelledWithoutCancel
	cpuSavedPerDay := totalCPUWithoutCancel - totalCPUWithCancel

	fmt.Printf("Service Parameters:\n")
	fmt.Printf("  Requests/day:       %d (10M)\n", requestsPerDay)
	fmt.Printf("  Cancel rate:        %.0f%%\n", cancelRate*100)
	fmt.Printf("  Cancelled/day:      %d (2M)\n", cancelledPerDay)
	fmt.Printf("  Full chain time:    %.0f ms\n", fullChainMs)
	fmt.Printf("  Cancel point:       %.0f ms (25%% into chain)\n", cancelledAtMs)
	fmt.Println()

	fmt.Printf("Daily CPU Time Comparison:\n")
	fmt.Printf("  Without cancellation: %.0f CPU-seconds/day\n", totalCPUWithoutCancel/1000)
	fmt.Printf("  With cancellation:    %.0f CPU-seconds/day\n", totalCPUWithCancel/1000)
	fmt.Printf("  CPU time saved:       %.0f CPU-seconds/day\n", cpuSavedPerDay/1000)
	fmt.Printf("  Reduction:            %.1f%%\n", (cpuSavedPerDay/totalCPUWithoutCancel)*100)
	fmt.Println()

	// AWS cost projection
	// t3.medium: 2 vCPU, $0.0416/vCPU-hour = $0.0832/hour
	// CPU-seconds to vCPU-hours: divide by 3600
	costPerVCPUHour := 0.0416
	vCPUHoursSaved := cpuSavedPerDay / 1000 / 3600 // CPU-seconds → hours
	dailySavings := vCPUHoursSaved * costPerVCPUHour
	monthlySavings := dailySavings * 30

	fmt.Printf("AWS Cost Impact (t3.medium @ $0.0416/vCPU-hour):\n")
	fmt.Printf("  vCPU-hours saved/day:   %.1f hours\n", vCPUHoursSaved)
	fmt.Printf("  Daily savings:          $%.2f\n", dailySavings)
	fmt.Printf("  Monthly savings:        $%.2f\n", monthlySavings)
	fmt.Printf("  Yearly savings:         $%.2f\n", monthlySavings*12)
	fmt.Println()

	// Instance reduction potential
	// t3.medium has 2 vCPU = 7200 CPU-seconds/hour capacity
	instanceCPUCapacityPerDay := 2.0 * 3600 * 24 // CPU-seconds per instance per day
	instancesFreed := cpuSavedPerDay / 1000 / instanceCPUCapacityPerDay
	instanceCostMonth := 30.0 // $30/month per t3.medium

	fmt.Printf("Instance Reduction Potential:\n")
	fmt.Printf("  CPU capacity freed:     %.2f instance-equivalents\n", instancesFreed)
	fmt.Printf("  Potential savings:      $%.2f/month (fewer instances needed)\n",
		instancesFreed*instanceCostMonth)
	fmt.Println()

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("  With 20%% early cancellation at 10M req/day:\n")
	fmt.Printf("    • Save %.0f CPU-seconds/day of wasted computation\n", cpuSavedPerDay/1000)
	fmt.Printf("    • Reduce CPU utilization by %.1f%%\n", (cpuSavedPerDay/totalCPUWithoutCancel)*100)
	fmt.Printf("    • Monthly savings: $%.2f in compute costs\n", monthlySavings)
	fmt.Printf("    • Key insight: cancelled requests should EXIT FAST, not run to completion\n")
}

// --- Main Demo ---

func main() {
	fmt.Println("=== Context Cancellation & Resource Cleanup Pattern ===")
	fmt.Println()

	// 1. Cascading cancellation demo
	fmt.Println("--- 1. Cascading Cancellation (HTTP → DB → Cache) ---")
	fmt.Println()

	// Normal completion
	ctx := context.Background()
	result := CascadingCall(ctx)
	fmt.Printf("  Normal (no cancel):  completed=%v, steps=%d, duration=%v\n",
		result.Completed, result.StepsRun, result.Duration.Round(time.Millisecond))

	// Cancel after HTTP call completes (~60ms into 160ms chain)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	result = CascadingCall(ctx)
	cancel()
	fmt.Printf("  Cancel at 60ms:      completed=%v, steps=%d, cancelled_at=%s, duration=%v\n",
		result.Completed, result.StepsRun, result.CancelledAt, result.Duration.Round(time.Millisecond))

	// Cancel immediately
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond) // ensure timeout fires
	result = CascadingCall(ctx)
	cancel()
	fmt.Printf("  Cancel immediately:  completed=%v, steps=%d, cancelled_at=%s, duration=%v\n",
		result.Completed, result.StepsRun, result.CancelledAt, result.Duration.Round(time.Millisecond))
	fmt.Println()

	// 2. No cancellation demo
	fmt.Println("--- 2. No Cancellation (runs to completion regardless) ---")
	fmt.Println()

	// Even with a cancelled context, NoCancellationCall ignores it
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	result = NoCancellationCall(ctx)
	cancel()
	fmt.Printf("  Cancelled context:   completed=%v, steps=%d, duration=%v (WASTED!)\n",
		result.Completed, result.StepsRun, result.Duration.Round(time.Millisecond))

	result = NoCancellationCall(context.Background())
	fmt.Printf("  Normal context:      completed=%v, steps=%d, duration=%v\n",
		result.Completed, result.StepsRun, result.Duration.Round(time.Millisecond))
	fmt.Println()

	// 3. Anti-pattern demo
	fmt.Println("--- 3. Anti-Pattern: context.Background() in Goroutine ---")
	fmt.Println()

	// Cancel parent after 60ms — good goroutine stops, bad one continues
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	goodResult, badResult := AntiPatternDemo(parentCtx)
	parentCancel()

	fmt.Printf("  Parent cancelled at 60ms:\n")
	fmt.Printf("    GOOD (parent ctx):     completed=%v, steps=%d, duration=%v\n",
		goodResult.Completed, goodResult.StepsRun, goodResult.Duration.Round(time.Millisecond))
	fmt.Printf("    BAD  (Background()):   completed=%v, steps=%d, duration=%v ← LEAKED WORK!\n",
		badResult.Completed, badResult.StepsRun, badResult.Duration.Round(time.Millisecond))
	fmt.Println()

	// 4. Early cancel simulation
	fmt.Println("--- 4. Early Cancel Simulation (20% cancel rate) ---")
	fmt.Println()

	numRequests := 100
	withCancel, withoutCancel := EarlyCancelDemo(numRequests, 0.20)

	fmt.Printf("  %d requests (20%% cancelled early):\n", numRequests)
	fmt.Printf("    With cancellation:    %v\n", withCancel.Round(time.Millisecond))
	fmt.Printf("    Without cancellation: %v\n", withoutCancel.Round(time.Millisecond))
	saved := withoutCancel - withCancel
	if saved > 0 {
		fmt.Printf("    Wall-clock saved:     %v (%.1f%% reduction)\n",
			saved.Round(time.Millisecond),
			float64(saved)/float64(withoutCancel)*100)
	}
	fmt.Println()

	// CPU time analysis
	fullChainMs := float64(totalChainLatency.Milliseconds())
	cancelPointMs := fullChainMs * 0.25
	cancelledRequests := int(float64(numRequests) * 0.20)
	cpuSavedMs := float64(cancelledRequests) * (fullChainMs - cancelPointMs)

	fmt.Printf("  CPU time analysis (per batch of %d requests):\n", numRequests)
	fmt.Printf("    Wasted CPU without cancel: %.0f ms (cancelled requests run full chain)\n",
		float64(cancelledRequests)*fullChainMs)
	fmt.Printf("    Actual CPU with cancel:    %.0f ms (cancelled requests exit at 25%%)\n",
		float64(cancelledRequests)*cancelPointMs)
	fmt.Printf("    CPU time saved:            %.0f ms\n", cpuSavedMs)
	fmt.Println()

	// 5. Cost projection
	fmt.Println()
	calculateCostProjection()
}
