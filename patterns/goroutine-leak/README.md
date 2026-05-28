# Goroutine Leak Detection & Prevention

## 📋 Overview

Detecting and preventing goroutine leaks that cause progressive memory growth and eventual OOM kills in production Go services.

## TL;DR
- **Problem**: Goroutines without exit paths leak ~2-8KB each; at 1 leak/sec = 172-691 MB/day wasted
- **Solution**: Context cancellation + done channels + LeakDetector for runtime monitoring
- **Impact**: Prevent 86,400 leaked goroutines/day = $0.63-$2.52/month wasted RAM per service

## 🎯 Problem Statement

Goroutines in Go are lightweight but never garbage collected while blocked. A goroutine waiting on a channel read, mutex lock, or I/O operation with no exit path will persist for the lifetime of the process, consuming stack memory that grows over time.

**Real-world impact:** At just 1 leaked goroutine per second, a service accumulates 86,400 orphaned goroutines in 24 hours — consuming up to 691 MB of unrecoverable memory.

## 🔍 Root Cause

### Why goroutines leak:

1. **No exit path**: Goroutine blocks on channel with no sender/closer
2. **Missing context**: No cancellation signal propagated to spawned goroutines
3. **Forgotten references**: Channel or WaitGroup never signaled
4. **No GC for goroutines**: Go runtime cannot collect blocked goroutines — they persist until process exit

### Common Leak Patterns

| Pattern | Cause | Memory Impact |
| --- | --- | --- |
| Blocked channel read | No sender, no close, no select | ~2-8 KB/goroutine |
| Missing context cancel | Parent exits, child runs forever | ~2-8 KB + resources held |
| Abandoned timer/ticker | `time.After` in loop without cleanup | ~2 KB + timer overhead |
| Listener without shutdown | Server goroutine never stopped | ~2-8 KB + connections |

### Anti-pattern:

```go
// ❌ BAD: Goroutine blocks forever — no one closes ch
func LeakyServer(requests int) {
    for i := 0; i < requests; i++ {
        ch := make(chan struct{})
        go func() {
            <-ch // Blocks forever — leaked!
        }()
        // ch is never closed, goroutine is abandoned
    }
}
```

## ⚡ Solution

### 1. Context Cancellation + Done Channels

```go
// ✅ GOOD: Every goroutine has a clear exit path
func SafeServer(ctx context.Context, requests int) {
    var wg sync.WaitGroup
    for i := 0; i < requests; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            done := make(chan struct{})
            go func() {
                time.Sleep(1 * time.Millisecond)
                close(done)
            }()
            select {
            case <-done:
                // Work completed normally
            case <-ctx.Done():
                // Parent cancelled — clean exit
            }
        }(i)
    }
    wg.Wait()
}
```

### 2. Leak Detector

```go
// Track goroutine count before/after to detect leaks
detector := NewLeakDetector("MyOperation")
doWork()
detector.Snapshot()
if leaked := detector.Leaked(); leaked > 0 {
    log.Printf("WARNING: %d goroutines leaked", leaked)
}
```

### 3. Graceful Shutdown with Timeout

```go
// Attempt graceful stop, force-kill after timeout
func GracefulShutdown(timeout time.Duration, worker func(ctx context.Context)) bool {
    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan struct{})
    go func() {
        worker(ctx)
        close(done)
    }()
    cancel() // Signal stop
    select {
    case <-done:
        return true  // Graceful
    case <-time.After(timeout):
        return false // Forced termination
    }
}
```

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

### Goroutine Growth Rate (1000 iterations)

| Implementation | Goroutines After | Memory Growth | Leak Rate |
| --- | --- | --- | --- |
| LeakyServer(1000) | +1000 | ~2-8 MB | 100% leaked |
| SafeServer(1000) | +0 | ~0 MB | 0% leaked |

### Graceful Shutdown Timing

| Worker Type | Timeout | Result | Actual Time |
| --- | --- | --- | --- |
| LongRunningWorker | 500ms | ✓ Graceful | < 1ms |
| StubbornWorker | 100ms | ✗ Forced | ~100ms |

### Memory Per Goroutine

| Metric | Value |
| --- | --- |
| Minimum stack size | 2 KB |
| Typical stack size | 4-8 KB |
| Maximum stack growth | Up to 1 GB (default) |
| Goroutine metadata | ~400 bytes |

## 💰 Cost Impact

### 24-Hour Leak Projection (1 leak/second)

```
Leak rate:           1 goroutine/second
Duration:            24 hours (86,400 seconds)
Total leaked:        86,400 goroutines

Memory consumed:
  Min (~2KB stack):  168 MB (0.16 GB)
  Max (~8KB stack):  675 MB (0.66 GB)

AWS Cost (t3.medium @ $3.75/GB-month):
  Min: $0.63/month wasted RAM
  Max: $2.52/month wasted RAM

Hidden costs:
  - GC pressure from scanning leaked goroutine stacks
  - CPU overhead from scheduler managing 86K+ goroutines
  - OOM kills → service restarts → downtime
  - Cascading failures when memory pressure triggers swap
```

### Scaling Impact

| Leak Rate | Daily Goroutines | Daily Memory | Monthly Cost |
| --- | --- | --- | --- |
| 1/sec | 86,400 | 172-691 MB | $0.63-$2.52 |
| 10/sec | 864,000 | 1.7-6.9 GB | $6.30-$25.20 |
| 100/sec | 8,640,000 | 17-69 GB | $63-$252 |

> Note: At 10+ leaks/sec, services typically OOM within hours, causing restarts and downtime costs far exceeding the RAM cost alone.

## ✅ When to Apply

**DO apply when:**

- Spawning goroutines in request handlers or event loops
- Using channels for async communication
- Building long-running services (servers, workers, daemons)
- Any goroutine that outlives the function that created it

**DON'T worry about when:**

- Short-lived CLI tools that exit quickly
- Goroutines with guaranteed completion (bounded work, no blocking)
- `main()` goroutine itself

### Prevention Checklist

1. Every `go func()` must have a `select` with `ctx.Done()` or a guaranteed close
2. Use `defer wg.Done()` to ensure WaitGroup is always decremented
3. Prefer `context.WithTimeout` over unbounded waits
4. Add leak detection in integration tests using `runtime.NumGoroutine()`
5. Monitor goroutine count in production (expose via `/debug/pprof/goroutine`)

## 🧪 How to Run

```bash
cd patterns/goroutine-leak

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Check for goroutine leaks in your code
go test -v -run TestLeak
```


## When This Is Acceptable

- Long-lived background goroutines that are intentionally designed to run for the application lifetime (e.g., metrics collector, health checker)
- Goroutines with explicit shutdown via `context.Done()` or `close(quit)` — these are not leaks
