package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	eh "github.com/alpardfm/cost-efficient-go/patterns/error-handling"
)

// globalErr prevents compiler from optimizing away error allocations.
var globalErr error

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
		globalErr = eh.ErrorsNew()
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
		globalErr = eh.FmtErrorf()
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
		globalErr = eh.SentinelError
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
		globalErr = eh.PreallocatedError
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
	s1, f1 := eh.HotPathWithErrors(hotPathIter, eh.ErrorsNew)
	d1 := time.Since(start)
	fmt.Printf("errors.New():     %v (success=%d, errors=%d)\n", d1, s1, f1)

	start = time.Now()
	s2, f2 := eh.HotPathWithErrors(hotPathIter, eh.FmtErrorf)
	d2 := time.Since(start)
	fmt.Printf("fmt.Errorf():     %v (success=%d, errors=%d)\n", d2, s2, f2)

	start = time.Now()
	s3, f3 := eh.HotPathWithSentinel(hotPathIter)
	d3 := time.Since(start)
	fmt.Printf("Sentinel:         %v (success=%d, errors=%d)\n", d3, s3, f3)

	start = time.Now()
	s4, f4 := eh.HotPathWithPreallocated(hotPathIter)
	d4 := time.Since(start)
	fmt.Printf("Preallocated:     %v (success=%d, errors=%d)\n", d4, s4, f4)
	fmt.Println()

	// Suppress unused variable warnings
	_ = rand.Int()

	// 3. Verify error interface compliance
	fmt.Println("--- Error Interface Compliance ---")
	var err error
	err = eh.ErrorsNew()
	fmt.Printf("errors.New():     %q\n", err.Error())
	err = eh.FmtErrorf()
	fmt.Printf("fmt.Errorf():     %q\n", err.Error())
	err = eh.SentinelError
	fmt.Printf("SentinelError:    %q\n", err.Error())
	err = eh.PreallocatedError
	fmt.Printf("PreallocatedErr:  %q\n", err.Error())
	fmt.Println()

	// 4. Cost projection
	calculateCostProjection()
}
