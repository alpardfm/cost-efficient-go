# Context Cancellation & Resource Cleanup

## 📋 Overview

Proper context propagation ensures cancelled requests exit fast instead of burning CPU on work nobody will consume.

## TL;DR
- **Problem**: Cancelled requests that run to completion waste CPU; services ignore context cancellation, burning compute on abandoned work
- **Solution**: Cascading cancellation (HTTP → DB → Cache) with `ctx.Done()` checks at each step; proper context propagation saves 15%+ compute at 20% cancel rate
- **Impact**: 20% cancel rate at 10M req/day saves ~$83/month in compute costs

## 🎯 Problem Statement

When a client disconnects or a timeout fires, services that ignore context cancellation continue burning CPU and memory on work that nobody will ever consume. In a typical microservice call chain (HTTP → DB → Cache), each step takes time. If the client disconnects after the first step, the remaining steps are pure waste.

**Real-world impact:** A service handling 10M requests/day with 20% early cancellation (client timeouts, load balancer cuts, user navigates away) wastes ~240,000 CPU-seconds/day on abandoned work!

## 🔍 Root Cause

### Why services waste resources on cancelled requests:

1. **No context checking**: Functions use `time.Sleep` or blocking I/O without checking `ctx.Done()`
2. **context.Background() in goroutines**: Spawned goroutines lose the parent's cancellation signal
3. **Fire-and-forget patterns**: Downstream calls proceed regardless of upstream state
4. **Missing cascading propagation**: Cancel signal doesn't flow through the entire call chain

### The waste chain:

```go
// ❌ BAD: Ignores context cancellation entirely
func HandleRequest(ctx context.Context) {
    // Client disconnects here... but we keep going!
    result := callHTTPService()     // 50ms wasted
    data := queryDatabase(result)   // 80ms wasted
    writeCache(data)                // 30ms wasted
    // Total: 160ms of CPU burned for nothing
}
```

### Anti-pattern: context.Background() in goroutines

```go
// ❌ BAD: Goroutine becomes "orphaned" — cannot be cancelled
func ProcessAsync(parentCtx context.Context) {
    go func() {
        // Anti-pattern: loses parent cancellation signal!
        ctx := context.Background()
        expensiveWork(ctx)  // Runs forever even if parent is cancelled
    }()
}
```

At 20% cancel rate on 10M requests/day:
- 2M requests run to completion unnecessarily
- Each wastes ~120ms of CPU time (remaining chain after cancel point)
- Total: **240,000 CPU-seconds/day** of pure waste

## ⚡ Solution

Implement cascading cancellation: propagate the context through every step and check `ctx.Done()` before proceeding. When the parent context is cancelled, all child operations exit immediately.

### Cascading Cancellation Pattern (GOOD)

```go
// ✅ GOOD: Each step checks context before proceeding
func CascadingCall(ctx context.Context) CallResult {
    // Step 1: HTTP downstream call
    if err := simulateWork(ctx, 50*time.Millisecond); err != nil {
        return CallResult{CancelledAt: "http_call"}
    }

    // Step 2: DB query
    if err := simulateWork(ctx, 80*time.Millisecond); err != nil {
        return CallResult{CancelledAt: "db_query"}
    }

    // Step 3: Cache write
    if err := simulateWork(ctx, 30*time.Millisecond); err != nil {
        return CallResult{CancelledAt: "cache_write"}
    }

    return CallResult{Completed: true}
}

// Context-aware work simulation
func simulateWork(ctx context.Context, duration time.Duration) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // Exit immediately on cancellation
    case <-time.After(duration):
        return nil
    }
}
```

### Proper Context Propagation in Goroutines

```go
// ✅ GOOD: Goroutine inherits parent context — cancellable
func ProcessAsync(parentCtx context.Context) {
    go func() {
        result := CascadingCall(parentCtx)  // Cancelled when parent is done
        // ...
    }()
}
```

### Early Cancel Simulation

```go
// Simulate client disconnect: cancel after 25% of chain time
ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
defer cancel()
result := CascadingCall(ctx)  // Exits at ~40ms instead of running full 160ms
```

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

### Cascading Cancellation vs No Cancellation

| **Scenario** | **Duration** | **Steps Run** | **CPU Saved** |
| --- | --- | --- | --- |
| Normal (no cancel) | ~160 ms | 3/3 | — |
| Cancel at 60ms (cascading) | ~60 ms | 1/3 | ~100 ms |
| Cancel immediately (cascading) | ~0 ms | 0/3 | ~160 ms |
| Cancel at 60ms (no cancellation) | ~160 ms | 3/3 | 0 ms (WASTED!) |

### Anti-Pattern Demonstration (parent cancelled at 60ms)

| **Implementation** | **Completed** | **Duration** | **Result** |
| --- | --- | --- | --- |
| GOOD (parent ctx) | No | ~60 ms | Properly cancelled |
| BAD (Background()) | Yes | ~160 ms | Leaked work! |

### Batch Simulation (100 requests, 20% cancel rate)

| **Mode** | **Wall-Clock Time** | **CPU Time Saved** |
| --- | --- | --- |
| With cancellation | ~160 ms | ~2,400 ms total CPU |
| Without cancellation | ~160 ms | 0 ms |

**CPU time analysis per batch of 100 requests:**
- Wasted CPU without cancel: 3,200 ms (20 cancelled requests × 160ms full chain)
- Actual CPU with cancel: 800 ms (20 cancelled requests × 40ms early exit)
- CPU time saved: **2,400 ms per 100 requests**

```bash
# Run benchmarks yourself
cd patterns/context-cancellation
go test -bench=. -benchmem -benchtime=3s
```

## 💰 Cost Impact

### Service Parameters

| **Parameter** | **Value** |
| --- | --- |
| Requests/day | 10,000,000 (10M) |
| Cancel rate | 20% |
| Cancelled/day | 2,000,000 (2M) |
| Full chain time | 160 ms |
| Cancel point | 40 ms (25% into chain) |

### Daily CPU Time Comparison

| **Mode** | **CPU-seconds/day** |
| --- | --- |
| Without cancellation | 1,600,000 |
| With cancellation | 1,360,000 |
| **CPU time saved** | **240,000** |
| **Reduction** | **15.0%** |

### AWS Cost Projection (t3.medium @ $0.0416/vCPU-hour)

| **Metric** | **Value** |
| --- | --- |
| vCPU-hours saved/day | 66.7 hours |
| Daily savings | $2.77 |
| **Monthly savings** | **$83.20** |
| Yearly savings | $998.40 |

### Instance Reduction Potential

```
CPU capacity freed:     ~1.4 instance-equivalents
Potential savings:      ~$42/month (fewer t3.medium instances needed)

Key insight: cancelled requests should EXIT FAST, not run to completion
```

### Scaling Impact

| **Cancel Rate** | **Monthly Savings** | **CPU Reduction** |
| --- | --- | --- |
| 10% | ~$41.60 | ~7.5% |
| 20% | ~$83.20 | ~15.0% |
| 30% | ~$124.80 | ~22.5% |

## When to Apply

✅ **DO apply when:**

- Service has multi-step call chains (HTTP → DB → Cache → etc.)
- Client disconnects are common (mobile apps, load balancer timeouts)
- Operations take 100ms+ total
- Service handles high throughput (1M+ req/day)
- You're spawning goroutines for downstream work
- Cancel rate is non-trivial (≥5% of requests)

❌ **DON'T over-optimize when:**

- Operations complete in < 1ms (overhead of context check not worth it)
- Work is idempotent and results are cached (might as well finish)
- Fire-and-forget operations that must complete regardless (audit logs, billing)
- Single-step operations with no downstream calls

### Decision Guide

```
Is this a multi-step operation? (2+ sequential calls)
├── YES → Does each step take 10ms+?
│         ├── YES → Add ctx.Done() check before each step
│         └── NO  → Check only before expensive steps
└── NO  → Is the single operation > 100ms?
          ├── YES → Add periodic ctx.Done() checks within the operation
          └── NO  → Context checking overhead not worth it
```

### Common Anti-Patterns to Avoid

| **Anti-Pattern** | **Fix** |
| --- | --- |
| `context.Background()` in goroutines | Pass parent context to goroutines |
| Ignoring `ctx.Err()` return value | Check and return early on cancellation |
| No timeout on downstream calls | Use `context.WithTimeout()` for all external calls |
| Blocking without select on `ctx.Done()` | Use `select` with `ctx.Done()` case |

## 🧪 How to Run

```bash
cd patterns/context-cancellation

# Run demo (shows cascading cancellation, anti-pattern, and cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Check context propagation with race detector
go run -race main.go
```


## When This Is Acceptable

- Fire-and-forget background tasks that must complete regardless of caller cancellation
- Cleanup/shutdown operations that should not be interrupted
- Operations with their own timeout that is shorter than the parent context
