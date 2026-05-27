// Package main demonstrates goroutine leak detection and prevention.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/goroutine-leak/examples/
package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	leak "github.com/alpardfm/cost-efficient-go/patterns/goroutine-leak"
)

func main() {
	fmt.Println("=== Goroutine Leak Detection & Prevention ===")
	fmt.Println()

	// 1. Demonstrate leak detection
	fmt.Println("--- 1. Leak Detection ---")
	fmt.Println()

	// Leaky implementation
	detector := leak.NewLeakDetector("LeakyServer")
	leak.LeakyServer(100)
	time.Sleep(10 * time.Millisecond) // Let goroutines settle
	detector.Snapshot()
	detector.Report()
	fmt.Println()

	// Safe implementation
	detector2 := leak.NewLeakDetector("SafeServer")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	leak.SafeServer(ctx, 100)
	cancel()
	time.Sleep(10 * time.Millisecond)
	detector2.Snapshot()
	detector2.Report()
	fmt.Println()

	// 2. Demonstrate graceful shutdown
	fmt.Println("--- 2. Graceful Shutdown ---")
	fmt.Println()

	// Well-behaved worker
	fmt.Print("  LongRunningWorker (responds to cancel): ")
	graceful := leak.GracefulShutdown(500*time.Millisecond, leak.LongRunningWorker)
	if graceful {
		fmt.Println("✓ Graceful shutdown")
	} else {
		fmt.Println("✗ Forced termination")
	}

	// Stubborn worker
	fmt.Print("  StubbornWorker (slow to respond):       ")
	graceful = leak.GracefulShutdown(100*time.Millisecond, leak.StubbornWorker)
	if graceful {
		fmt.Println("✓ Graceful shutdown")
	} else {
		fmt.Println("✗ Forced termination")
	}
	fmt.Println()

	// 3. Demonstrate growth rate
	fmt.Println("--- 3. Goroutine Growth Rate ---")
	fmt.Println()
	baseCount := runtime.NumGoroutine()
	fmt.Printf("  Base goroutine count: %d\n", baseCount)

	for _, n := range []int{10, 50, 100, 500} {
		before := runtime.NumGoroutine()
		leak.LeakyServer(n)
		runtime.Gosched()
		after := runtime.NumGoroutine()
		fmt.Printf("  After LeakyServer(%d): +%d goroutines (total: %d)\n",
			n, after-before, after)
	}
	fmt.Println()

	// 4. Cost projection
	fmt.Println("--- 4. Cost Impact ---")
	fmt.Println()
	leak.CostProjection24h()
}
