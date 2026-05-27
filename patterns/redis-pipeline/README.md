# Redis Pipeline & Connection Efficiency

## TL;DR
- **Problem**: Individual Redis ops = 1 round-trip each; N commands × 0.5ms network latency = N × 0.5ms total wait
- **Solution**: Pipeline batches N commands in 1 round-trip → 50-100x faster; Lua scripting for atomic server-side execution
- **Impact**: 10M cache ops/day, pipeline reduces latency by 80%, saves ~$24.5/month on smaller ElastiCache instance

## Problem

Every individual Redis GET/SET incurs a full network round-trip:

```
Client → SET key:0 → Redis → OK → Client    (0.5ms)
Client → SET key:1 → Redis → OK → Client    (0.5ms)
Client → SET key:2 → Redis → OK → Client    (0.5ms)
...
Client → SET key:99 → Redis → OK → Client   (0.5ms)

Total: 100 commands × 0.5ms = 50ms just in network wait
```

At scale with 10M cache operations/day:
- **5M seconds** of cumulative network wait (individual ops)
- Connection held longer per request → pool exhaustion under load
- Higher p99 latency → cascading timeouts in microservice chains
- Larger ElastiCache instance needed to handle connection count

## Root Cause

Redis is single-threaded and processes commands sequentially. The bottleneck isn't Redis execution time (sub-microsecond per command) — it's the **network round-trip** between client and server:

1. **Per-command overhead**: Each command requires a TCP send + wait + receive cycle
2. **Connection hold time**: Client holds a pool connection for N round-trips instead of 1
3. **Syscall overhead**: Each round-trip = 2 syscalls (write + read) on the client
4. **Head-of-line blocking**: Subsequent commands wait for previous round-trips to complete

The fundamental issue: commands that could be batched are sent individually, multiplying network latency by the number of operations.

## Solution

### 1. Pipeline — Batch Commands in Single Round-Trip

```go
// Before: N commands = N round-trips
for i := 0; i < numOps; i++ {
    mock.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
}

// After: N commands = 1 round-trip
cmds := make([]PipelineCommand, numOps)
for i := 0; i < numOps; i++ {
    cmds[i] = PipelineCommand{
        Op:    "SET",
        Key:   fmt.Sprintf("key:%d", i),
        Value: fmt.Sprintf("value:%d", i),
    }
}
mock.ExecPipeline(cmds)
```

Pipeline sends all commands at once and reads all responses at once — single round-trip regardless of N.

### 2. Lua Scripting — Atomic Server-Side Execution

```go
// Before: check-then-set = 2 round-trips per key
for _, key := range keys {
    _, exists := mock.Get(key)
    if !exists {
        mock.Set(key, value)
    }
}
// N keys × 2 round-trips = 2N round-trips

// After: Lua script = 1 round-trip for all keys
mock.ExecLuaScript("SET_IF_NOT_EXISTS", keys, values)
// 1 round-trip total, atomic execution
```

Lua scripts execute atomically on the Redis server — no intermediate round-trips, no race conditions.

### 3. Connection Pool Sizing

```go
// Pool acts as semaphore limiting concurrent connections
pool := make(chan struct{}, poolSize)

// Too small: goroutines block waiting for connections
// Too large: wasted memory, Redis connection limit hit
// Optimal: matches peak concurrent request count
```

Pool size directly impacts throughput under concurrent load. Undersized pools create contention; oversized pools waste resources.

## Benchmarks

> Machine: Apple M1, Go 1.24.4

### Individual vs Pipeline (1ms simulated latency)

| Operations | Individual | Pipeline | Speedup |
|-----------:|-----------:|---------:|--------:|
| 10 | ~20ms | ~2ms | **10x** |
| 50 | ~100ms | ~2ms | **50x** |
| 100 | ~200ms | ~2ms | **100x** |

Pipeline speedup scales linearly with operation count: N ops → Nx faster.

### Lua Script vs Individual Operations

| Keys | Individual (check+set) | Lua Script | Speedup |
|-----:|-----------------------:|-----------:|--------:|
| 10 | ~20ms | ~1ms | **20x** |
| 50 | ~100ms | ~1ms | **100x** |
| 100 | ~200ms | ~1ms | **200x** |

Lua eliminates both the per-command round-trip AND the race condition between check and set.

### Connection Pool Size Impact (20 workers, 500μs latency)

| Pool Size | Total Time | Throughput (ops/sec) |
|----------:|-----------:|---------------------:|
| 1 | ~200ms | ~2,000 |
| 5 | ~45ms | ~8,900 |
| 10 | ~25ms | ~16,000 |
| 20 | ~15ms | ~26,700 |
| 50 | ~12ms | ~33,300 |

Diminishing returns beyond pool size matching concurrency level.

### Range Enforcement

All benchmarks enforce 10-100 operations (clamped):
- Input < 10 → clamped to 10 (minimum)
- Input > 100 → clamped to 100 (maximum)

*Note: Run `go test -bench=. -benchmem` in `patterns/redis-pipeline/` for exact numbers on your machine.*

## Cost Impact

### Latency Savings Per Request

```
Service: 10M cache operations/day
Average: 5 Redis commands per API request → 2M requests/day
Network latency: 0.5ms per round-trip (same AZ)

Individual: 5 ops × 0.5ms = 2.5ms per request
Pipeline:   1 round-trip × 0.5ms = 0.5ms per request
Saved:      2.0ms per request (80% reduction)
```

### Connection Pool Efficiency (at 1000 req/sec peak)

```
Individual: holds connection for 2.5ms → needs ~3 connections
Pipeline:   holds connection for 0.5ms → needs ~1 connection
Reduction:  80% fewer connections needed
```

### AWS ElastiCache Cost

| Scenario | Instance | Monthly Cost |
|----------|----------|-------------:|
| Individual ops (more connections) | cache.t3.medium | ~$49/month |
| Pipeline ops (fewer connections) | cache.t3.small | ~$24.5/month |
| **Monthly savings** | | **$24.5/month ($294/year)** |

### Syscall Savings

```
Individual: 10M ops × 2 syscalls = 20M syscalls/day
Pipeline:   2M requests × 2 syscalls = 4M syscalls/day
Saved:      16M syscalls/day (~16 CPU-seconds/day)
```

### Total Estimated Savings

```
ElastiCache downsize:  $24.5/month
CPU (reduced syscalls): ~$0.01/month
Total:                 ~$24.5/month ($294/year)
Primary benefit:       Lower p99 latency, not just cost
```

## When to Apply

### ✅ DO use Pipeline when:
- Multiple Redis commands are independent (no command depends on previous result)
- Batch size is known ahead of time (10-100+ commands)
- Latency is the bottleneck (network-bound, not CPU-bound)
- High-throughput service with many cache operations per request

### ✅ DO use Lua scripting when:
- Multiple commands must execute atomically (check-then-set, increment-then-read)
- Server-side logic eliminates intermediate round-trips
- Race conditions between commands are a concern

### ✅ DO tune pool size when:
- Concurrent request count exceeds current pool size
- Connection wait time appears in latency metrics
- Redis `INFO clients` shows connection churn

### ❌ DON'T pipeline when:
- Commands depend on previous results (use Lua instead)
- Only 1-2 commands per request (overhead not worth it)
- Pipeline batch is unbounded (memory risk on client side)

### ❌ DON'T over-size the pool when:
- Redis `maxclients` limit would be exceeded
- Each connection consumes significant memory
- Actual concurrency is much lower than pool size

## How to Run

```bash
cd patterns/redis-pipeline

# Run demo (shows individual vs pipeline, pool sizing, Lua scripts, cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Run benchmarks with longer duration for stable results
go test -bench=. -benchmem -benchtime=5s

# Run only pipeline comparison benchmarks
go test -bench=BenchmarkIndividual -benchmem
go test -bench=BenchmarkPipeline -benchmem

# Run pool size benchmarks
go test -bench=BenchmarkPool -benchmem

# Run Lua script benchmarks
go test -bench=BenchmarkLua -benchmem
```
