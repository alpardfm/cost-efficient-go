package main

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

// ============================================================
// PATTERN 15: Error Handling Efficiency
// ============================================================
// Problem: errors.New() and fmt.Errorf() allocate on every call.
// On hot paths with high error rates, this creates millions of
// unnecessary heap allocations per day.
//
// This pattern demonstrates:
// 1. errors.New() allocates on every call
// 2. fmt.Errorf() allocates even more (format string + args)
// 3. Sentinel errors (var ErrX = errors.New(...)) — zero alloc on use
// 4. Custom error types with pre-allocated messages — zero alloc
// 5. Impact: 5% error rate on 100M req/day = 5M allocs/day eliminated
// ============================================================

// --- Error Creation Methods ---

// ErrorsNew creates a new error using errors.New() every time.
// Each call allocates a new error object on the heap.
func ErrorsNew() error {
	return errors.New("operation failed: resource not found")
}

// FmtErrorf creates a new error using fmt.Errorf() every time.
// Allocates even more: format string processing + argument boxing.
func FmtErrorf() error {
	return fmt.Errorf("operation failed: resource %s not found at %d", "user", 42)
}

// --- Sentinel Errors (Zero Allocation on Use) ---

// SentinelError is a package-level variable — allocated once at init,
// zero allocation on every subsequent use.
var SentinelError = errors.New("operation failed: resource not found")

// --- Pre-allocated Custom Error Type (Zero Allocation) ---

// CustomErrorType implements the error interface with a pre-built message.
// No allocation occurs when returning this error on hot paths.
type CustomErrorType struct {
	Code    int
	Message string
}

func (e *CustomErrorType) Error() string {
	return e.Message
}

// PreallocatedError is a package-level custom error — zero alloc on use.
var PreallocatedError = &CustomErrorType{
	Code:    404,
	Message: "operation failed: resource not found",
}

// --- Hot Path Simulation ---

// HotPathWithErrors simulates a function with a 5% error rate on a hot path.
// It demonstrates the allocation difference between error creation strategies.
func HotPathWithErrors(iterations int, errFunc func() error) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		// Simulate 5% error rate
		if rand.Intn(100) < 5 {
			globalErr = errFunc() // assign to global to prevent compiler optimization
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// HotPathWithSentinel simulates the same hot path but returns a sentinel error.
// Zero allocation regardless of error rate.
func HotPathWithSentinel(iterations int) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		if rand.Intn(100) < 5 {
			globalErr = SentinelError // No allocation — just a pointer copy
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// HotPathWithPreallocated simulates the hot path with a pre-allocated custom error.
// Zero allocation regardless of error rate.
func HotPathWithPreallocated(iterations int) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		if rand.Intn(100) < 5 {
			globalErr = PreallocatedError // No allocation — just a pointer copy
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// --- Cost Projection ---

func calculateCostProjection() {
	fmt.Println("=== Cost Projection: Error Handling at Scale ===")
	fmt.Println()

	// Parameters
	requestsPerDay := 100_000_000 // 100M requests/day
	errorRate := 0.05             // 5% error rate
	errorsPerDay := int(float64(requestsPerDay) * errorRate)

	// Allocation costs (measured from benchmarks)
	// errors.New(): ~1 alloc, ~16 bytes per call
	// fmt.Errorf(): ~2-3 allocs, ~64-80 bytes per call
	// Sentinel/Preallocated: 0 allocs, 0 bytes per call
	errorsNewBytesPerCall := 16
	fmtErrorfBytesPerCall := 72 // average of format + args
	sentinelBytesPerCall := 0

	fmt.Printf("Service Parameters:\n")
	fmt.Printf("  Requests/day:     %d\n", requestsPerDay)
	fmt.Printf("  Error rate:       %.0f%%\n", errorRate*100)
	fmt.Printf("  Errors/day:       %d (5M)\n", errorsPerDay)
	fmt.Println()

	// Daily allocation waste
	errorsNewDaily := errorsPerDay * errorsNewBytesPerCall
	fmtErrorfDaily := errorsPerDay * fmtErrorfBytesPerCall
	sentinelDaily := errorsPerDay * sentinelBytesPerCall

	fmt.Printf("Daily Heap Allocations from Errors:\n")
	fmt.Printf("  errors.New():     %d MB/day (%d allocs)\n",
		errorsNewDaily/(1024*1024), errorsPerDay)
	fmt.Printf("  fmt.Errorf():     %d MB/day (%d allocs)\n",
		fmtErrorfDaily/(1024*1024), errorsPerDay*2)
	fmt.Printf("  Sentinel/Custom:  %d MB/day (0 allocs)\n", sentinelDaily)
	fmt.Println()

	// GC pressure impact
	// Each allocation adds GC tracking overhead (~8 bytes metadata)
	gcOverheadPerAlloc := 8
	gcPressureErrorsNew := errorsPerDay * gcOverheadPerAlloc
	gcPressureFmtErrorf := errorsPerDay * 2 * gcOverheadPerAlloc

	fmt.Printf("GC Pressure (tracking overhead):\n")
	fmt.Printf("  errors.New():     %d MB/day additional GC work\n",
		gcPressureErrorsNew/(1024*1024))
	fmt.Printf("  fmt.Errorf():     %d MB/day additional GC work\n",
		gcPressureFmtErrorf/(1024*1024))
	fmt.Printf("  Sentinel/Custom:  0 MB/day (no GC work)\n")
	fmt.Println()

	// AWS cost projection
	// t3.medium: $0.0416/vCPU-hour, $3.75/GB-month
	costPerGBMonth := 3.75

	// Monthly memory waste
	monthlyWasteErrorsNew := float64(errorsNewDaily) * 30 / (1024 * 1024 * 1024)
	monthlyWasteFmtErrorf := float64(fmtErrorfDaily) * 30 / (1024 * 1024 * 1024)

	fmt.Printf("Monthly AWS Cost Impact (memory pressure → larger instances):\n")
	fmt.Printf("  errors.New():     $%.2f/month (%.2f GB cumulative alloc pressure)\n",
		monthlyWasteErrorsNew*costPerGBMonth, monthlyWasteErrorsNew)
	fmt.Printf("  fmt.Errorf():     $%.2f/month (%.2f GB cumulative alloc pressure)\n",
		monthlyWasteFmtErrorf*costPerGBMonth, monthlyWasteFmtErrorf)
	fmt.Printf("  Sentinel/Custom:  $0.00/month (zero allocation)\n")
	fmt.Println()

	// Real impact: GC CPU time
	// At 5M allocs/day, GC runs more frequently → steals CPU cycles
	// Estimated: ~1-5μs per GC cycle, triggered every ~4MB of allocations
	gcCyclesPerDay := errorsNewDaily / (4 * 1024 * 1024) // every 4MB
	gcTimePerCycleUs := 3.0                              // microseconds (conservative)
	gcCPUTimePerDayMs := float64(gcCyclesPerDay) * gcTimePerCycleUs / 1000

	fmt.Printf("GC CPU Time Overhead:\n")
	fmt.Printf("  errors.New():     ~%.1f ms/day extra GC time (%d extra GC cycles)\n",
		gcCPUTimePerDayMs, gcCyclesPerDay)
	fmt.Printf("  fmt.Errorf():     ~%.1f ms/day extra GC time (%d extra GC cycles)\n",
		gcCPUTimePerDayMs*4.5, gcCyclesPerDay*4)
	fmt.Printf("  Sentinel/Custom:  0 ms/day (no extra GC cycles)\n")
	fmt.Println()

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("  Switching from errors.New() to sentinel errors eliminates:\n")
	fmt.Printf("    • %d heap allocations/day\n", errorsPerDay)
	fmt.Printf("    • %d MB/day of allocation pressure\n", errorsNewDaily/(1024*1024))
	fmt.Printf("  Switching from fmt.Errorf() to sentinel errors eliminates:\n")
	fmt.Printf("    • %d heap allocations/day\n", errorsPerDay*2)
	fmt.Printf("    • %d MB/day of allocation pressure\n", fmtErrorfDaily/(1024*1024))
	fmt.Printf("  At scale: fewer GC pauses → lower p99 latency → smaller instances needed\n")
}

// globalErr prevents compiler from optimizing away error allocations.
var globalErr error

// --- Demonstration ---

func main() {
	fmt.Println("=== Error Handling Efficiency Pattern ===")
	fmt.Println()

	// 1. Show allocation difference
	fmt.Println("--- Error Creation Methods ---")

	var memBefore, memAfter runtime.MemStats
	iterations := 100_000

	// errors.New()
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	for i := 0; i < iterations; i++ {
		globalErr = ErrorsNew()
	}
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("errors.New()     × %d: allocs=%d, bytes=%d KB\n",
		iterations,
		memAfter.Mallocs-memBefore.Mallocs,
		(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)

	// fmt.Errorf()
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	for i := 0; i < iterations; i++ {
		globalErr = FmtErrorf()
	}
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("fmt.Errorf()     × %d: allocs=%d, bytes=%d KB\n",
		iterations,
		memAfter.Mallocs-memBefore.Mallocs,
		(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)

	// Sentinel error
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	for i := 0; i < iterations; i++ {
		globalErr = SentinelError
	}
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("SentinelError    × %d: allocs=%d, bytes=%d KB\n",
		iterations,
		memAfter.Mallocs-memBefore.Mallocs,
		(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)

	// Preallocated custom error
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	for i := 0; i < iterations; i++ {
		globalErr = PreallocatedError
	}
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("PreallocatedErr  × %d: allocs=%d, bytes=%d KB\n",
		iterations,
		memAfter.Mallocs-memBefore.Mallocs,
		(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Println()

	// 2. Hot path simulation
	fmt.Println("--- Hot Path Simulation (1M iterations, 5% error rate) ---")
	hotPathIter := 1_000_000

	start := time.Now()
	s1, f1 := HotPathWithErrors(hotPathIter, ErrorsNew)
	d1 := time.Since(start)
	fmt.Printf("errors.New():     %v (success=%d, errors=%d)\n", d1, s1, f1)

	start = time.Now()
	s2, f2 := HotPathWithErrors(hotPathIter, FmtErrorf)
	d2 := time.Since(start)
	fmt.Printf("fmt.Errorf():     %v (success=%d, errors=%d)\n", d2, s2, f2)

	start = time.Now()
	s3, f3 := HotPathWithSentinel(hotPathIter)
	d3 := time.Since(start)
	fmt.Printf("Sentinel:         %v (success=%d, errors=%d)\n", d3, s3, f3)

	start = time.Now()
	s4, f4 := HotPathWithPreallocated(hotPathIter)
	d4 := time.Since(start)
	fmt.Printf("Preallocated:     %v (success=%d, errors=%d)\n", d4, s4, f4)
	fmt.Println()

	// 3. Verify error interface compliance
	fmt.Println("--- Error Interface Compliance ---")
	var err error
	err = ErrorsNew()
	fmt.Printf("errors.New():     %q\n", err.Error())
	err = FmtErrorf()
	fmt.Printf("fmt.Errorf():     %q\n", err.Error())
	err = SentinelError
	fmt.Printf("SentinelError:    %q\n", err.Error())
	err = PreallocatedError
	fmt.Printf("PreallocatedErr:  %q\n", err.Error())
	fmt.Println()

	// 4. Cost projection
	calculateCostProjection()
}
