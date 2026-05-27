package main

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================
// PATTERN 20: Redis Pipeline & Connection Efficiency
// ============================================================
// Problem: Individual Redis GET/SET operations incur a network
// round-trip per command. At scale, this latency dominates total
// request time and wastes connection resources.
//
// This pattern demonstrates:
// 1. Individual ops: one round-trip per command (slow)
// 2. Pipeline ops: batch commands in single round-trip (fast)
// 3. Connection pool sizing: impact on concurrent throughput
// 4. Lua scripting: atomic multi-command execution server-side
// 5. Impact: 10M cache ops/day → significant latency savings
//
// NOTE: Uses in-memory RedisMock for reproducible benchmarks.
// No real Redis instance required.
// ============================================================

// --- Redis Mock Implementation ---

// RedisMock simulates a Redis server with configurable network latency.
// It supports GET, SET, pipeline, and Lua script execution.
type RedisMock struct {
	mu       sync.RWMutex
	data     map[string]string
	latency  time.Duration // Simulated network round-trip latency
	poolSize int
}

// NewRedisMock creates a new RedisMock with the given latency and pool size.
func NewRedisMock(latency time.Duration, poolSize int) *RedisMock {
	return &RedisMock{
		data:     make(map[string]string),
		latency:  latency,
		poolSize: poolSize,
	}
}

// simulateRoundTrip simulates network latency for a single round-trip.
func (r *RedisMock) simulateRoundTrip() {
	if r.latency > 0 {
		time.Sleep(r.latency)
	}
}

// Set stores a key-value pair (simulates one round-trip).
func (r *RedisMock) Set(key, value string) {
	r.simulateRoundTrip()
	r.mu.Lock()
	r.data[key] = value
	r.mu.Unlock()
}

// Get retrieves a value by key (simulates one round-trip).
// Returns the value and whether the key exists.
func (r *RedisMock) Get(key string) (string, bool) {
	r.simulateRoundTrip()
	r.mu.RLock()
	val, ok := r.data[key]
	r.mu.RUnlock()
	return val, ok
}

// PipelineCommand represents a single command in a pipeline.
type PipelineCommand struct {
	Op    string // "SET" or "GET"
	Key   string
	Value string // Used for SET only
}

// PipelineResult holds the result of a single pipeline command.
type PipelineResult struct {
	Value string
	Found bool
}

// ExecPipeline executes multiple commands in a single round-trip.
// This is the key optimization: N commands, 1 round-trip instead of N.
func (r *RedisMock) ExecPipeline(commands []PipelineCommand) []PipelineResult {
	// Single round-trip for the entire batch
	r.simulateRoundTrip()

	results := make([]PipelineResult, len(commands))
	r.mu.Lock()
	for i, cmd := range commands {
		switch cmd.Op {
		case "SET":
			r.data[cmd.Key] = cmd.Value
			results[i] = PipelineResult{Value: "OK", Found: true}
		case "GET":
			val, ok := r.data[cmd.Key]
			results[i] = PipelineResult{Value: val, Found: ok}
		}
	}
	r.mu.Unlock()
	return results
}

// ExecLuaScript executes a Lua-like script atomically on the server side.
// This simulates Redis EVAL — multiple operations in one round-trip with
// server-side logic (no intermediate client round-trips).
func (r *RedisMock) ExecLuaScript(script string, keys []string, args []string) []PipelineResult {
	// Single round-trip for the entire script execution
	r.simulateRoundTrip()

	results := make([]PipelineResult, 0, len(keys))
	r.mu.Lock()
	defer r.mu.Unlock()

	switch script {
	case "GET_MULTI":
		// Lua script equivalent: return multiple GETs atomically
		for _, key := range keys {
			val, ok := r.data[key]
			results = append(results, PipelineResult{Value: val, Found: ok})
		}
	case "SET_IF_NOT_EXISTS":
		// Lua script equivalent: SET multiple keys only if they don't exist
		for i, key := range keys {
			if _, exists := r.data[key]; !exists {
				value := ""
				if i < len(args) {
					value = args[i]
				}
				r.data[key] = value
				results = append(results, PipelineResult{Value: "OK", Found: true})
			} else {
				results = append(results, PipelineResult{Value: "", Found: false})
			}
		}
	case "INCREMENT_MULTI":
		// Lua script equivalent: increment counters atomically
		for _, key := range keys {
			val, ok := r.data[key]
			if !ok {
				val = "0"
			}
			// Simple string-based increment for simulation
			num := 0
			fmt.Sscanf(val, "%d", &num)
			num++
			r.data[key] = fmt.Sprintf("%d", num)
			results = append(results, PipelineResult{Value: r.data[key], Found: true})
		}
	}
	return results
}

// --- Individual Operations (Before: Slow) ---

// IndividualOps performs GET and SET operations one by one.
// Each operation incurs a separate network round-trip.
// For N operations, this costs N round-trips.
func IndividualOps(mock *RedisMock, numOps int) time.Duration {
	if numOps < 10 {
		numOps = 10
	}
	if numOps > 100 {
		numOps = 100
	}

	start := time.Now()

	// SET operations one by one
	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		mock.Set(key, value)
	}

	// GET operations one by one
	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("key:%d", i)
		globalVal, globalFound = mock.Get(key)
	}

	return time.Since(start)
}

// --- Pipeline Operations (After: Fast) ---

// PipelineOps performs GET and SET operations in batched pipelines.
// All commands are sent in a single round-trip per batch.
// For N operations, this costs 2 round-trips (1 for SETs, 1 for GETs).
func PipelineOps(mock *RedisMock, numOps int) time.Duration {
	if numOps < 10 {
		numOps = 10
	}
	if numOps > 100 {
		numOps = 100
	}

	start := time.Now()

	// Batch all SET commands into one pipeline
	setCmds := make([]PipelineCommand, numOps)
	for i := 0; i < numOps; i++ {
		setCmds[i] = PipelineCommand{
			Op:    "SET",
			Key:   fmt.Sprintf("key:%d", i),
			Value: fmt.Sprintf("value:%d", i),
		}
	}
	mock.ExecPipeline(setCmds)

	// Batch all GET commands into one pipeline
	getCmds := make([]PipelineCommand, numOps)
	for i := 0; i < numOps; i++ {
		getCmds[i] = PipelineCommand{
			Op:  "GET",
			Key: fmt.Sprintf("key:%d", i),
		}
	}
	results := mock.ExecPipeline(getCmds)

	// Use last result to prevent compiler optimization
	if len(results) > 0 {
		globalVal = results[len(results)-1].Value
		globalFound = results[len(results)-1].Found
	}

	return time.Since(start)
}

// --- Pool Size Demo ---

// PoolSizeDemo demonstrates the impact of connection pool size on throughput
// under concurrent access. Smaller pools create contention; larger pools
// allow more parallel operations.
func PoolSizeDemo(poolSize int, concurrency int, opsPerWorker int) (time.Duration, int) {
	if opsPerWorker < 10 {
		opsPerWorker = 10
	}
	if opsPerWorker > 100 {
		opsPerWorker = 100
	}

	mock := NewRedisMock(500*time.Microsecond, poolSize)

	// Semaphore to simulate pool size limiting concurrent connections
	pool := make(chan struct{}, poolSize)
	for i := 0; i < poolSize; i++ {
		pool <- struct{}{}
	}

	var wg sync.WaitGroup
	totalOps := 0
	var mu sync.Mutex

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ops := 0
			for i := 0; i < opsPerWorker; i++ {
				// Acquire connection from pool
				<-pool
				key := fmt.Sprintf("worker:%d:key:%d", workerID, i)
				mock.Set(key, "data")
				// Release connection back to pool
				pool <- struct{}{}
				ops++
			}
			mu.Lock()
			totalOps += ops
			mu.Unlock()
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)
	return elapsed, totalOps
}

// --- Lua Script Demo ---

// LuaScriptDemo demonstrates Redis Lua scripting as an alternative to
// multiple round-trips. A Lua script executes atomically on the server,
// eliminating intermediate round-trips for multi-step operations.
func LuaScriptDemo(mock *RedisMock, numKeys int) (individualTime, luaTime time.Duration) {
	if numKeys < 10 {
		numKeys = 10
	}
	if numKeys > 100 {
		numKeys = 100
	}

	keys := make([]string, numKeys)
	values := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("lua:key:%d", i)
		values[i] = fmt.Sprintf("value:%d", i)
	}

	// Approach 1: Individual SET-if-not-exists (check + set = 2 round-trips each)
	mockIndividual := NewRedisMock(mock.latency, mock.poolSize)
	start := time.Now()
	for i := 0; i < numKeys; i++ {
		// Check if exists (1 round-trip)
		_, exists := mockIndividual.Get(keys[i])
		if !exists {
			// Set if not exists (1 round-trip)
			mockIndividual.Set(keys[i], values[i])
		}
	}
	individualTime = time.Since(start)

	// Approach 2: Lua script — single round-trip for all SET-if-not-exists
	mockLua := NewRedisMock(mock.latency, mock.poolSize)
	start = time.Now()
	mockLua.ExecLuaScript("SET_IF_NOT_EXISTS", keys, values)
	luaTime = time.Since(start)

	return individualTime, luaTime
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
	// Individual: holds connection for N round-trips
	// Pipeline: holds connection for 1 round-trip (releases faster)
	connHoldTimeIndividual := individualLatencyPerReq // ms per request
	connHoldTimePipeline := pipelineLatencyPerReq     // ms per request

	// Connections needed to handle peak load (assume 1000 req/sec peak)
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
	// Fewer connections = smaller Redis instance needed
	// ElastiCache pricing: cache.t3.medium = $0.068/hour = ~$49/month
	// cache.t3.small = $0.034/hour = ~$24.5/month
	costLargeInstance := 49.0 // $/month (needed for individual ops)
	costSmallInstance := 24.5 // $/month (sufficient for pipeline ops)
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
	// Each round-trip = 1 syscall (write) + 1 syscall (read)
	syscallsIndividual := opsPerDay * 2
	syscallsPipeline := (opsPerDay / avgOpsPerRequest) * 2 // batched
	syscallsSaved := syscallsIndividual - syscallsPipeline

	// Estimated CPU cost per syscall: ~1μs
	cpuTimeSavedSec := float64(syscallsSaved) / 1_000_000
	cpuCostPerHour := 0.0416 // t3.medium vCPU-hour
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

// Global variables to prevent compiler optimization
var (
	globalVal   string
	globalFound bool
)

// --- Demonstration ---

func main() {
	fmt.Println("=== Redis Pipeline & Connection Efficiency Pattern ===")
	fmt.Println()

	// Use 1ms simulated latency for visible demo results
	latency := 1 * time.Millisecond

	// 1. Individual vs Pipeline comparison
	fmt.Println("--- Individual vs Pipeline Operations ---")
	for _, numOps := range []int{10, 50, 100} {
		mock := NewRedisMock(latency, 10)

		individualTime := IndividualOps(mock, numOps)

		// Reset mock for fair comparison
		mock2 := NewRedisMock(latency, 10)
		pipelineTime := PipelineOps(mock2, numOps)

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
		elapsed, totalOps := PoolSizeDemo(poolSize, 20, 20)
		throughput := float64(totalOps) / elapsed.Seconds()
		fmt.Printf("  Pool=%2d: %v total, %d ops, %.0f ops/sec\n",
			poolSize, elapsed.Round(time.Millisecond), totalOps, throughput)
	}
	fmt.Println()

	// 3. Lua script demo
	fmt.Println("--- Lua Script vs Individual Operations ---")
	for _, numKeys := range []int{10, 50, 100} {
		mock := NewRedisMock(latency, 10)
		indivTime, luaTime := LuaScriptDemo(mock, numKeys)
		speedup := float64(indivTime) / float64(luaTime)
		fmt.Printf("  %3d keys: Individual=%v, Lua=%v (%.1fx faster)\n",
			numKeys, indivTime.Round(time.Microsecond),
			luaTime.Round(time.Microsecond), speedup)
	}
	fmt.Println()

	// 4. Verify correctness: pipeline produces same results as individual
	fmt.Println("--- Correctness Verification ---")
	mock := NewRedisMock(0, 10) // No latency for correctness check
	numOps := 20

	// Individual SET
	for i := 0; i < numOps; i++ {
		mock.Set(fmt.Sprintf("verify:%d", i), fmt.Sprintf("val:%d", i))
	}

	// Pipeline GET to verify
	getCmds := make([]PipelineCommand, numOps)
	for i := 0; i < numOps; i++ {
		getCmds[i] = PipelineCommand{Op: "GET", Key: fmt.Sprintf("verify:%d", i)}
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
	mockRange := NewRedisMock(0, 10)
	// Try with 5 ops (should be clamped to 10)
	IndividualOps(mockRange, 5)
	fmt.Printf("  Input=5 ops → clamped to 10 (minimum)\n")
	// Try with 200 ops (should be clamped to 100)
	IndividualOps(mockRange, 200)
	fmt.Printf("  Input=200 ops → clamped to 100 (maximum)\n")
	fmt.Println()

	// 6. Show pipeline with mixed operations
	fmt.Println("--- Mixed Pipeline (SET + GET in single batch) ---")
	mockMixed := NewRedisMock(latency, 10)
	mixedCmds := make([]PipelineCommand, 20)
	for i := 0; i < 10; i++ {
		mixedCmds[i] = PipelineCommand{Op: "SET", Key: fmt.Sprintf("mix:%d", i), Value: fmt.Sprintf("data:%d", i)}
	}
	for i := 0; i < 10; i++ {
		mixedCmds[10+i] = PipelineCommand{Op: "GET", Key: fmt.Sprintf("mix:%d", i)}
	}
	mixedResults := mockMixed.ExecPipeline(mixedCmds)
	fmt.Printf("  20 mixed commands (10 SET + 10 GET) in 1 round-trip\n")
	fmt.Printf("  Last GET result: key=mix:9, value=%q, found=%v\n",
		mixedResults[19].Value, mixedResults[19].Found)
	fmt.Println()

	// 7. Cost projection
	calculateCostProjection()

}
