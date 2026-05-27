// Package main demonstrates sync.Pool buffer reuse vs naive allocation.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/sync-pool/examples/
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	syncpool "github.com/alpardfm/cost-efficient-go/patterns/sync-pool"
)

func main() {
	fmt.Println("=== sync.Pool — Memory Pooling for Buffer Reuse ===")
	fmt.Println()

	// 1. Demonstrate naive vs pooled allocation
	fmt.Println("--- Buffer Allocation: Naive vs Pooled ---")
	const bufSize = 4096 // 4KB — typical HTTP response buffer

	pool := syncpool.NewBufferPool(bufSize)

	// Naive approach
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	const ops = 100_000
	for i := 0; i < ops; i++ {
		syncpool.GlobalSink = syncpool.NaiveBufferAlloc(bufSize)
	}

	runtime.ReadMemStats(&memAfter)
	naiveAllocs := memAfter.TotalAlloc - memBefore.TotalAlloc
	naiveNumAllocs := memAfter.Mallocs - memBefore.Mallocs
	fmt.Printf("  Naive (%d ops, %d KB buffer):\n", ops, bufSize/1024)
	fmt.Printf("    Total allocated: %d MB\n", naiveAllocs/(1024*1024))
	fmt.Printf("    Num allocations: %d\n", naiveNumAllocs)
	fmt.Println()

	// Pooled approach
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	for i := 0; i < ops; i++ {
		syncpool.GlobalSink = syncpool.PooledBufferAlloc(pool)
	}

	runtime.ReadMemStats(&memAfter)
	pooledAllocs := memAfter.TotalAlloc - memBefore.TotalAlloc
	pooledNumAllocs := memAfter.Mallocs - memBefore.Mallocs
	fmt.Printf("  Pooled (%d ops, %d KB buffer):\n", ops, bufSize/1024)
	fmt.Printf("    Total allocated: %d MB\n", pooledAllocs/(1024*1024))
	fmt.Printf("    Num allocations: %d\n", pooledNumAllocs)
	fmt.Println()

	if naiveAllocs > pooledAllocs {
		reduction := float64(naiveAllocs-pooledAllocs) / float64(naiveAllocs) * 100
		fmt.Printf("  Allocation reduction: %.1f%%\n", reduction)
	} else {
		fmt.Println("  Note: Pool overhead visible at this scale; benefits show in benchmarks")
	}
	fmt.Println()

	// 2. GC pressure comparison
	fmt.Println("--- GC Pressure Comparison ---")
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	gcBefore := memBefore.NumGC

	for i := 0; i < ops; i++ {
		syncpool.GlobalSink = syncpool.NaiveBufferAlloc(bufSize)
	}

	runtime.ReadMemStats(&memAfter)
	naiveGC := memAfter.NumGC - gcBefore
	fmt.Printf("  Naive:  %d GC cycles during %d ops\n", naiveGC, ops)

	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	gcBefore = memBefore.NumGC

	for i := 0; i < ops; i++ {
		syncpool.GlobalSink = syncpool.PooledBufferAlloc(pool)
	}

	runtime.ReadMemStats(&memAfter)
	pooledGC := memAfter.NumGC - gcBefore
	fmt.Printf("  Pooled: %d GC cycles during %d ops\n", pooledGC, ops)
	if naiveGC > 0 {
		fmt.Printf("  GC reduction: %.0f%%\n", float64(naiveGC-pooledGC)/float64(naiveGC)*100)
	}
	fmt.Println()

	// 3. Small object demo
	smallObjectDemo()

	// 4. Cost projection at scale
	fmt.Println("--- Cost Projection (4KB buffer per request) ---")
	fmt.Println()

	scales := []int{1_000_000, 10_000_000, 100_000_000}
	for _, scale := range scales {
		cp := &syncpool.CostProjection{
			RequestsPerDay:    scale,
			BufferSize:        bufSize,
			NaiveAllocPerReq:  int64(bufSize), // allocates full buffer each time
			PooledAllocPerReq: 64,             // amortized: occasional pool miss
		}
		calculate(cp)
	}

	// 5. Summary
	fmt.Println("--- Summary ---")
	fmt.Println("  ✅ Use sync.Pool when:")
	fmt.Println("     - Object size > 64 bytes (ideally > 1KB)")
	fmt.Println("     - Allocation frequency > 10K/sec")
	fmt.Println("     - Objects have uniform size")
	fmt.Println("     - GC pressure is measurable bottleneck")
	fmt.Println()
	fmt.Println("  ❌ Don't use sync.Pool when:")
	fmt.Println("     - Objects are small (< 64 bytes)")
	fmt.Println("     - Allocation rate is low")
	fmt.Println("     - Objects have variable sizes (pool fragmentation)")
	fmt.Println("     - Object initialization is complex (pool.New overhead)")
}

// smallObjectDemo demonstrates that sync.Pool is NOT effective for
// objects smaller than ~64 bytes.
func smallObjectDemo() {
	fmt.Println("--- When NOT to Use sync.Pool (Small Objects < 64 bytes) ---")
	fmt.Printf("  SmallObject size: %d bytes\n", 8) // int32 + int32 = 8 bytes
	fmt.Println()

	// Naive: allocate small object every time
	const iterations = 1_000_000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		syncpool.GlobalSmallSink = &syncpool.SmallObject{ID: int32(i), Value: int32(i * 2)}
	}
	naiveDuration := time.Since(start)

	// Pooled: use sync.Pool for small objects
	smallPool := sync.Pool{
		New: func() interface{} {
			return &syncpool.SmallObject{}
		},
	}

	start = time.Now()
	for i := 0; i < iterations; i++ {
		obj := smallPool.Get().(*syncpool.SmallObject)
		obj.ID = int32(i)
		obj.Value = int32(i * 2)
		syncpool.GlobalSmallSink = obj
		smallPool.Put(obj)
	}
	pooledDuration := time.Since(start)

	fmt.Printf("  Naive allocation:  %v (%d iterations)\n", naiveDuration, iterations)
	fmt.Printf("  Pooled allocation: %v (%d iterations)\n", pooledDuration, iterations)
	fmt.Println()

	if pooledDuration > naiveDuration {
		fmt.Println("  ⚠️  Pool is SLOWER for small objects!")
		fmt.Println("  Reason: interface{} boxing + pool mutex overhead > allocation cost")
	} else {
		fmt.Println("  Pool is faster here, but the margin is minimal for small objects.")
		fmt.Println("  The overhead/benefit ratio is poor compared to larger buffers.")
	}
	fmt.Println()
	fmt.Println("  Rule of thumb: Use sync.Pool when object size > 64 bytes")
	fmt.Println("  and allocation frequency > 10K/sec")
}

// calculate prints cost projection for a given configuration.
func calculate(cp *syncpool.CostProjection) {
	const (
		costPerGBMonth  = 3.75   // AWS t3.medium RAM cost
		costPerVCPUHour = 0.0416 // AWS t3.medium vCPU cost
		hoursPerMonth   = 730
	)

	naiveBytesPerDay := int64(cp.RequestsPerDay) * cp.NaiveAllocPerReq
	pooledBytesPerDay := int64(cp.RequestsPerDay) * cp.PooledAllocPerReq
	savedBytesPerDay := naiveBytesPerDay - pooledBytesPerDay

	savedGBPerDay := float64(savedBytesPerDay) / (1024 * 1024 * 1024)
	memorySavingsMonth := savedGBPerDay * costPerGBMonth

	// GC CPU savings: less garbage = less GC work
	gcWorkReductionSec := float64(savedBytesPerDay) / (1024 * 1024 * 1024) / 86400
	cpuSavingsMonth := gcWorkReductionSec * 3600 * hoursPerMonth * costPerVCPUHour

	fmt.Printf("  Scale: %dM requests/day\n", cp.RequestsPerDay/1_000_000)
	fmt.Printf("  Buffer size: %d KB\n", cp.BufferSize/1024)
	fmt.Printf("  Naive memory/day:  %.2f GB\n", float64(naiveBytesPerDay)/(1024*1024*1024))
	fmt.Printf("  Pooled memory/day: %.2f GB (reused, not newly allocated)\n", float64(pooledBytesPerDay)/(1024*1024*1024))
	fmt.Printf("  Memory saved/day:  %.2f GB\n", savedGBPerDay)
	fmt.Printf("  Monthly savings:   $%.2f (memory) + $%.4f (GC CPU) = $%.2f\n",
		memorySavingsMonth, cpuSavingsMonth, memorySavingsMonth+cpuSavingsMonth)
	fmt.Println()
}
