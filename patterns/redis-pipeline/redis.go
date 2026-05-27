package redis_pipeline

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
		GlobalVal, GlobalFound = mock.Get(key)
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
		GlobalVal = results[len(results)-1].Value
		GlobalFound = results[len(results)-1].Found
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

// Global variables to prevent compiler optimization
var (
	GlobalVal   string
	GlobalFound bool
)
