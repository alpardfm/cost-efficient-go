# Channel Patterns & Performance Trade-offs

## 📋 Overview

Choosing the right channel pattern for goroutine communication — unbuffered vs buffered vs mutex — to minimize blocking, reduce scheduling overhead, and maximize throughput.

## TL;DR
- **Problem**: Unbuffered channels block on every send/recv, forcing goroutine scheduling overhead on each operation
- **Solution**: Buffered channels with optimal size (100+), fan-out/fan-in for parallelism, mutex for simple shared state
- **Impact**: Buffered(100) is 3-4x faster than unbuffered for producer-consumer; at 100K ops/sec saves ~$0.30/month in CPU (small, but latency impact is bigger)

## 🎯 Problem Statement

Go channels are the idiomatic way to communicate between goroutines, but the wrong channel pattern creates hidden bottlenecks. Unbuffered channels force a synchronous handoff — every send blocks until a receiver is ready, and every receive blocks until a sender is ready. This tight coupling means the Go scheduler must context-switch goroutines on **every single message**, creating massive overhead in producer-consumer workloads.

**Real-world impact:** A microservice processing 100K messages/sec with unbuffered channels wastes ~4.8 CPU-hours/day on scheduling overhead alone.

## 🔍 Root Cause

### Why unbuffered channels are slow:

1. **Goroutine scheduling on every send**: The sender goroutine is parked, the receiver is woken up — this involves the Go scheduler on every operation
2. **No batching of work**: Each message requires a full context switch (save/restore goroutine state)
3. **Cache thrashing**: Frequent goroutine switches invalidate CPU cache lines, reducing effective throughput
4. **Lock contention**: Channel internals use a mutex; unbuffered channels hold this lock longer due to synchronous handoff

### The bottleneck:

```go
// ❌ SLOW: Every send blocks until receiver reads
ch := make(chan int)  // unbuffered

go func() {
    for val := range ch {
        process(val)  // Receiver must be here for sender to proceed
    }
}()

for i := 0; i < 100000; i++ {
    ch <- i  // BLOCKS on every iteration — scheduler involved each time
}
```

With unbuffered channels at high throughput:
- ~200 ns overhead per send/recv (goroutine park + wake)
- At 100K ops/sec = 20ms/sec of pure scheduling overhead
- CPU spends time switching goroutines instead of doing useful work

## ⚡ Solution

### 1. Buffered Channels (Primary Fix)

```go
// ✅ FAST: Producer can burst up to 100 messages without blocking
ch := make(chan int, 100)  // buffered

go func() {
    for val := range ch {
        process(val)
    }
}()

for i := 0; i < 100000; i++ {
    ch <- i  // Only blocks when buffer is full — amortized scheduling
}
```

Buffered channels amortize scheduling cost: the sender only blocks when the buffer is full, allowing burst writes without context switches.

### 2. Mutex-Based Alternative (For Simple Shared State)

```go
// ✅ FASTEST for accumulator patterns: no channel overhead at all
var mu sync.Mutex
var sum int64

// Producer appends under lock
mu.Lock()
for i := 0; i < messageCount; i++ {
    data = append(data, i)
}
mu.Unlock()
```

When you don't need communication semantics (just shared state), a mutex eliminates channel overhead entirely.

### 3. Fan-Out/Fan-In (For CPU-Bound Parallelism)

```go
// ✅ Maximize CPU utilization across all cores
func FanOutFanIn(items []int, numWorkers int) []int {
    jobs := make(chan int, len(items))
    results := make(chan int, len(items))

    // Fan-out: start workers
    var wg sync.WaitGroup
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for item := range jobs {
                results <- processItem(item)
            }
        }()
    }

    // Send all jobs
    for _, item := range items {
        jobs <- item
    }
    close(jobs)

    // Wait and close results
    go func() {
        wg.Wait()
        close(results)
    }()

    // Fan-in: collect results
    output := make([]int, 0, len(items))
    for result := range results {
        output = append(output, result)
    }
    return output
}
```

Fan-out/fan-in scales linearly up to GOMAXPROCS workers for CPU-bound work.

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

### Producer-Consumer Throughput (100K messages)

| **Pattern** | **Duration** | **~ns/op** | **Speedup vs Unbuffered** |
| --- | --- | --- | --- |
| Unbuffered | ~20ms | ~200 | 1.0x |
| Buffered(1) | ~18ms | ~180 | ~1.1x |
| Buffered(10) | ~10ms | ~100 | ~2.0x |
| Buffered(100) | ~5-6ms | ~50-60 | **~3-4x** |
| Buffered(1000) | ~5ms | ~50 | ~4x |
| Mutex-based | ~3ms | ~30 | ~6x |

### Fan-Out/Fan-In (10K items, CPU-bound work)

| **Pattern** | **Workers** | **Speedup** |
| --- | --- | --- |
| Sequential | 1 | 1.0x |
| Fan-out/fan-in | 8 (GOMAXPROCS) | ~4-6x |

### Buffer Size Impact

| **Buffer Size** | **Relative Speed** | **Notes** |
| --- | --- | --- |
| 1 | ~1.1x | Barely better than unbuffered |
| 10 | ~2x | Noticeable improvement |
| 100 | ~3-4x | Sweet spot for most workloads |
| 1000 | ~4x | Diminishing returns, more memory |

```bash
# Run benchmarks yourself
cd patterns/channel-patterns
go test -bench=. -benchmem -benchtime=3s
```

## 💰 Cost Impact

### Per-Operation Overhead (from benchmarks)

| **Pattern** | **ns/op** | **Scheduling Events** |
| --- | --- | --- |
| Unbuffered | ~200 ns | Every send + recv |
| Buffered(100) | ~50 ns | Only when buffer full/empty |
| Mutex-based | ~30 ns | No scheduling, just lock |

### Daily CPU Time at 100K ops/sec

| **Pattern** | **CPU-hours/day** | **Cost/day** | **Cost/month** |
| --- | --- | --- | --- |
| Unbuffered | ~4.8 hrs | $0.20 | $6.00 |
| Buffered(100) | ~1.2 hrs | $0.05 | $1.50 |
| Mutex-based | ~0.7 hrs | $0.03 | $0.90 |

### Monthly Savings vs Unbuffered (AWS t3.medium, $0.0416/vCPU-hour)

| **Switch To** | **Savings/month** | **Real Benefit** |
| --- | --- | --- |
| Buffered(100) | ~$0.30 | Lower p99 latency, fewer context switches |
| Mutex-based | ~$0.40 | Lowest overhead for shared state patterns |

> **Note:** The raw CPU cost savings are modest (~$0.30/month). The real value is in **latency reduction** — fewer goroutine context switches means lower p99 latency and better cache locality, which can defer instance scaling at high load.

### Multi-Core Contention Impact

```
At 100K ops/sec on 8 cores:
  • Unbuffered: scheduling overhead = ~2% of CPU time
  • Buffered(100): reduces scheduling by 4x → better cache locality → lower p99
  • At scale: fewer context switches = more useful work per CPU cycle
```

## When to Apply

### Channels vs Mutex Decision Guide

```
Do goroutines need to COMMUNICATE (send data)?
├── YES → Use channels
│         ├── High throughput (>10K msg/sec)? → Buffered (size 64-256)
│         ├── Low throughput or signaling? → Unbuffered is fine
│         └── Need parallel processing? → Fan-out/fan-in
└── NO → Just sharing state?
          ├── Simple counter/accumulator → sync.Mutex or sync/atomic
          └── Read-heavy, write-rare → sync.RWMutex
```

### ✅ Use buffered channels when:

- Producer-consumer pattern with high message rate
- You want to decouple producer/consumer speeds
- Burst traffic is expected (buffer absorbs spikes)
- Multiple producers or consumers

### ✅ Use unbuffered channels when:

- You need synchronization guarantees (sender knows receiver got the message)
- Signaling between goroutines (done channels, cancellation)
- Low-frequency communication where overhead doesn't matter
- Backpressure is desired (slow consumer should slow producer)

### ✅ Use mutex when:

- Simple shared state (counters, accumulators, caches)
- No communication semantics needed
- Maximum raw throughput on shared data
- Read-heavy workloads (use `sync.RWMutex`)

### ❌ Don't over-optimize when:

- Channel throughput is < 1K msg/sec (overhead is negligible)
- Code clarity would suffer significantly
- The bottleneck is I/O, not channel operations

### Buffer Size Guidelines

| **Throughput** | **Recommended Buffer** | **Rationale** |
| --- | --- | --- |
| < 1K msg/sec | Unbuffered or 1 | Overhead doesn't matter |
| 1K-10K msg/sec | 10-50 | Moderate improvement |
| 10K-100K msg/sec | 64-256 | Sweet spot |
| > 100K msg/sec | 256-1024 | Diminishing returns above this |

## 🧪 How to Run

```bash
cd patterns/channel-patterns

# Run demo (shows producer-consumer comparison, fan-out/fan-in, and cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Profile CPU usage to see scheduling overhead
go test -bench=BenchmarkUnbuffered -cpuprofile=cpu.prof
go tool pprof cpu.prof
```


## When This Is Acceptable

- Signal channels (`chan struct{}`, `chan bool`, `chan error`) used for done/notify patterns — these MUST be unbuffered
- Channels passed to `context.WithCancel`, `os/signal.Notify`, or similar coordination primitives
- Low-throughput control channels (< 100 ops/sec) where scheduling overhead is negligible
