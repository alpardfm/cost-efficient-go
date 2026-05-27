// Package main demonstrates Redis pipeline vs individual operations.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/redis-pipeline/examples/
package main

import (
	"fmt"
	"time"

	redis "github.com/alpardfm/cost-efficient-go/patterns/redis-pipeline"
)

func main() {
	fmt.Println("=== Redis Pipeline & Connection Efficiency Pattern ===")
	fmt.Println()

	// Use 1ms simulated latency for visible demo results
	latency := 1 * time.Millisecond

	// 1. Individual vs Pipeline comparison
	fmt.Println("--- Individual vs Pipeline Operations ---")
	for _, numOps := range []int{10, 50, 100} {
		mock := redis.NewRedisMock(latency, 10)

		individualTime := redis.IndividualOps(mock, numOps)

		// Reset mock for fair comparison
		mock2 := redis.NewRedisMock(latency, 10)
		pipelineTime := redis.PipelineOps(mock2, numOps)

		speedup := float64(individualTime) / float64(pipelineTime)
		fmt.Printf("  %3d ops: Individual=%v, Pipeline=%v (%.1fx faster)\n",
			numOps, individualTime.Round(time.Microsecond),
			pipelineTime.Round(time.Microsecond), speedup)
	}
	fmt.Println()

	// 2. Pool size impact
	fmt.Println("--- Connection Pool Size Impact ---")
	fmt.Printf("  (20 concurrent workers, 20 ops each, 500μs latency)\n")
	for _, poolSize := range []int{1, 5, 10, 20, 50} {
		elapsed, totalOps := redis.PoolSizeDemo(poolSize, 20, 20)
		throughput := float64(totalOps) / elapsed.Seconds()
		fmt.Printf("  Pool=%2d: %v total, %d ops, %.0f ops/sec\n",
			poolSize, elapsed.Round(time.Millisecond), totalOps, throughput)
	}
	fmt.Println()

	// 3. Lua script demo
	fmt.Println("--- Lua Script vs Individual Operations ---")
	for _, numKeys := range []int{10, 50, 100} {
		mock := redis.NewRedisMock(latency, 10)
		indivTime, luaTime := redis.LuaScriptDemo(mock, numKeys)
		speedup := float64(indivTime) / float64(luaTime)
		fmt.Printf("  %3d keys: Individual=%v, Lua=%v (%.1fx faster)\n",
			numKeys, indivTime.Round(time.Microsecond),
			luaTime.Round(time.Microsecond), speedup)
	}
	fmt.Println()

	// 4. Verify correctness: pipeline produces same results as individual
	fmt.Println("--- Correctness Verification ---")
	mock := redis.NewRedisMock(0, 10) // No latency for correctness check
	numOps := 20

	// Individual SET
	for i := 0; i < numOps; i++ {
		mock.Set(fmt.Sprintf("verify:%d", i), fmt.Sprintf("val:%d", i))
	}

	// Pipeline GET to verify
	getCmds := make([]redis.PipelineCommand, numOps)
	for i := 0; i < numOps; i++ {
		getCmds[i] = redis.PipelineCommand{Op: "GET", Key: fmt.Sprintf("verify:%d", i)}
	}
	results := mock.ExecPipeline(getCmds)

	allCorrect := true
	for i, r := range results {
		expected := fmt.Sprintf("val:%d", i)
		if r.Value != expected || !r.Found {
			allCorrect = false
			fmt.Printf("  MISMATCH at key %d: got=%q, want=%q\n", i, r.Value, expected)
		}
	}
	if allCorrect {
		fmt.Printf("  ✓ All %d operations verified correct (pipeline == individual)\n", numOps)
	}
	fmt.Println()

	// 5. Show range enforcement
	fmt.Println("--- Range Enforcement (10-100 ops) ---")
	mockRange := redis.NewRedisMock(0, 10)
	// Try with 5 ops (should be clamped to 10)
	redis.IndividualOps(mockRange, 5)
	fmt.Printf("  Input=5 ops → clamped to 10 (minimum)\n")
	// Try with 200 ops (should be clamped to 100)
	redis.IndividualOps(mockRange, 200)
	fmt.Printf("  Input=200 ops → clamped to 100 (maximum)\n")
	fmt.Println()

	// 6. Show pipeline with mixed operations
	fmt.Println("--- Mixed Pipeline (SET + GET in single batch) ---")
	mockMixed := redis.NewRedisMock(latency, 10)
	mixedCmds := make([]redis.PipelineCommand, 20)
	for i := 0; i < 10; i++ {
		mixedCmds[i] = redis.PipelineCommand{Op: "SET", Key: fmt.Sprintf("mix:%d", i), Value: fmt.Sprintf("data:%d", i)}
	}
	for i := 0; i < 10; i++ {
		mixedCmds[10+i] = redis.PipelineCommand{Op: "GET", Key: fmt.Sprintf("mix:%d", i)}
	}
	mixedResults := mockMixed.ExecPipeline(mixedCmds)
	fmt.Printf("  20 mixed commands (10 SET + 10 GET) in 1 round-trip\n")
	fmt.Printf("  Last GET result: key=mix:9, value=%q, found=%v\n",
		mixedResults[19].Value, mixedResults[19].Found)
	fmt.Println()

	// 7. Cost projection
	calculateCostProjection()
}

// --- Cost Projection ---

func calculateCostProjection() {
	fmt.Println("=== Cost Projection: Redis Pipeline at Scale ===")
	fmt.Println()

	// Parameters
	opsPerDay := 10_000_000 // 10M cache operations/day
	avgOpsPerRequest := 5   // Average Redis commands per API request
	requestsPerDay := opsPerDay / avgOpsPerRequest

	// Latency parameters (typical Redis over network)
	networkLatencyMs := 0.5 // 0.5ms per round-trip (same AZ)

	// Individual: each op = 1 round-trip
	individualLatencyPerReq := float64(avgOpsPerRequest) * networkLatencyMs
	// Pipeline: all ops = 1 round-trip
	pipelineLatencyPerReq := networkLatencyMs

	fmt.Printf("Service Parameters:\n")
	fmt.Printf("  Cache operations/day:  %d (10M)\n", opsPerDay)
	fmt.Printf("  Avg ops per request:   %d\n", avgOpsPerRequest)
	fmt.Printf("  Requests/day:          %d (2M)\n", requestsPerDay)
	fmt.Printf("  Network latency:       %.1fms per round-trip\n", networkLatencyMs)
	fmt.Println()

	// Latency savings
	latencySavedPerReq := individualLatencyPerReq - pipelineLatencyPerReq
	totalLatencySavedPerDay := latencySavedPerReq * float64(requestsPerDay) / 1000 // seconds

	fmt.Printf("Latency Impact:\n")
	fmt.Printf("  Individual ops:  %.1fms per request (%d round-trips)\n",
		individualLatencyPerReq, avgOpsPerRequest)
	fmt.Printf("  Pipeline ops:    %.1fms per request (1 round-trip)\n",
		pipelineLatencyPerReq)
	fmt.Printf("  Savings:         %.1fms per request (%.0f%% reduction)\n",
		latencySavedPerReq, (latencySavedPerReq/individualLatencyPerReq)*100)
	fmt.Printf("  Daily savings:   %.0f seconds of cumulative wait time\n",
		totalLatencySavedPerDay)
	fmt.Println()

	// Connection efficiency
	connHoldTimeIndividual := individualLatencyPerReq
	connHoldTimePipeline := pipelineLatencyPerReq

	peakRPS := 1000.0
	connsNeededIndividual := peakRPS * connHoldTimeIndividual / 1000
	connsNeededPipeline := peakRPS * connHoldTimePipeline / 1000

	fmt.Printf("Connection Pool Efficiency (at %d req/sec peak):\n", int(peakRPS))
	fmt.Printf("  Individual ops:  %.0f connections needed\n", connsNeededIndividual)
	fmt.Printf("  Pipeline ops:    %.0f connections needed\n", connsNeededPipeline)
	fmt.Printf("  Reduction:       %.0f%% fewer connections\n",
		(1-connsNeededPipeline/connsNeededIndividual)*100)
	fmt.Println()

	// AWS cost impact
	costLargeInstance := 49.0
	costSmallInstance := 24.5
	monthlySavings := costLargeInstance - costSmallInstance

	fmt.Printf("AWS ElastiCache Cost Impact:\n")
	fmt.Printf("  Individual ops (more connections): cache.t3.medium ~$%.0f/month\n",
		costLargeInstance)
	fmt.Printf("  Pipeline ops (fewer connections):  cache.t3.small  ~$%.1f/month\n",
		costSmallInstance)
	fmt.Printf("  Monthly savings:                   $%.1f/month ($%.0f/year)\n",
		monthlySavings, monthlySavings*12)
	fmt.Println()

	// CPU savings from reduced syscalls
	opsPerDay2 := 10_000_000
	avgOpsPerRequest2 := 5
	syscallsIndividual := opsPerDay2 * 2
	syscallsPipeline := (opsPerDay2 / avgOpsPerRequest2) * 2
	syscallsSaved := syscallsIndividual - syscallsPipeline

	cpuTimeSavedSec := float64(syscallsSaved) / 1_000_000
	cpuCostPerHour := 0.0416
	cpuSavingsMonth := (cpuTimeSavedSec / 3600) * cpuCostPerHour * 30

	fmt.Printf("CPU Savings (reduced syscalls):\n")
	fmt.Printf("  Individual ops:  %dM syscalls/day\n", syscallsIndividual/1_000_000)
	fmt.Printf("  Pipeline ops:    %dM syscalls/day\n", syscallsPipeline/1_000_000)
	fmt.Printf("  Saved:           %dM syscalls/day (%.1f CPU-seconds/day)\n",
		syscallsSaved/1_000_000, cpuTimeSavedSec)
	fmt.Printf("  Monthly savings: $%.4f/month (CPU time)\n", cpuSavingsMonth)
	fmt.Println()

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("  Pipeline reduces per-request Redis latency by %.0f%%\n",
		(latencySavedPerReq/individualLatencyPerReq)*100)
	fmt.Printf("  Connection pool needs reduced by %.0f%%\n",
		(1-connsNeededPipeline/connsNeededIndividual)*100)
	fmt.Printf("  Total estimated savings: $%.1f/month ($%.0f/year)\n",
		monthlySavings+cpuSavingsMonth, (monthlySavings+cpuSavingsMonth)*12)
	fmt.Printf("  Primary benefit: lower p99 latency, not just cost\n")
}
