package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	worker_pool "github.com/alpardfm/cost-efficient-go/patterns/worker-pool"
)

func main() {
	fmt.Println("=== Worker Pool Pattern ===")
	fmt.Println()

	// Generate tasks
	tasks := make([]worker_pool.Task, 1000)
	for i := range tasks {
		tasks[i] = worker_pool.Task{
			ID:       i,
			Duration: time.Duration(rand.Intn(5)) * time.Millisecond,
		}
	}

	// 1. Unbounded goroutines
	fmt.Println("--- Unbounded Goroutines (1000 tasks) ---")
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	worker_pool.ProcessUnbounded(tasks)
	unboundedDuration := time.Since(start)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("Time: %v\n", unboundedDuration)
	fmt.Printf("Goroutines spawned: %d\n", len(tasks))
	fmt.Printf("Memory allocated: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Println()

	// 2. Worker pool with different sizes
	fmt.Println("--- Worker Pool (1000 tasks, varying workers) ---")
	for _, workers := range []int{4, 8, 16, 32, 64} {
		start := time.Now()
		worker_pool.ProcessWithPool(tasks, workers)
		duration := time.Since(start)
		fmt.Printf("  workers=%2d → %v\n", workers, duration)
	}
	fmt.Println()

	// 3. errgroup.SetLimit comparison
	fmt.Println("--- errgroup.SetLimit (1000 tasks, varying workers) ---")
	for _, workers := range []int{4, 8, 16, 32, 64} {
		start := time.Now()
		worker_pool.ProcessWithErrgroup(tasks, workers)
		duration := time.Since(start)
		fmt.Printf("  limit=%2d → %v\n", workers, duration)
	}
	fmt.Println()

	// 4. Goroutine count comparison
	fmt.Println("--- Peak Goroutine Count ---")
	fmt.Printf("  Unbounded: %d goroutines (1 per task)\n", len(tasks))
	fmt.Printf("  Pool(16):  %d goroutines (fixed)\n", 16)
	fmt.Printf("  Memory per goroutine: ~2-8 KB\n")
	fmt.Printf("  Unbounded 1M tasks: ~2-8 GB goroutine stacks!\n")
	fmt.Printf("  Pool(16) 1M tasks:  ~32-128 KB goroutine stacks\n")
}
