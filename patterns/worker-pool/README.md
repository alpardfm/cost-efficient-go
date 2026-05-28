# Worker Pool Pattern

## 📋 Overview

Controlling concurrency with a fixed worker pool instead of spawning unbounded goroutines — preventing memory explosion and upstream overload.

## TL;DR
- **Problem**: Unbounded goroutines cause memory explosion (1M tasks = 4GB stacks), CPU thrashing, and upstream overload
- **Solution**: Fixed worker pool with buffered channel for backpressure, or `errgroup.SetLimit()` for simpler cases
- **Impact**: 99.9% less goroutine memory, 10x fewer allocations; can downgrade from r5.xlarge ($200/mo) to t3.medium ($30/mo)

## 🎯 Problem Statement

Spawning one goroutine per task seems easy in Go, but at scale:

- **Memory explosion** — each goroutine uses 2-8 KB stack. 1M tasks = 2-8 GB just for stacks
- **CPU thrashing** — too many goroutines competing for CPU time
- **Upstream overload** — 1000 concurrent DB/HTTP connections overwhelm the target
- **No backpressure** — system accepts work faster than it can process

**Real-world impact:** A batch job processing 100K items with unbounded goroutines can OOM-kill your service.

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4:

### CPU-Bound Tasks (100 tasks, no I/O)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Unbounded (100 goroutines) | 209,991 | 10,513 | **202** |
| Pool 8 workers | 238,678 | **3,780** | **11** |
| Pool 16 workers | **169,983** | **3,910** | **19** |

**Pool 16: 19% faster, 63% less memory, 10x fewer allocations** than unbounded.

### I/O-Bound Tasks (100 tasks, 1ms I/O each)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Unbounded (100 goroutines) | **1,650,185** | 20,113 | **302** |
| Pool 8 workers | 21,338,824 | 4,550 | 19 |
| Pool 16 workers | 8,340,421 | 5,440 | 35 |
| Pool 32 workers | 5,726,606 | 7,241 | 67 |

**For I/O-bound: unbounded is faster** (all 100 tasks run simultaneously). But at the cost of 100 concurrent connections to upstream.

### Scale: 1000 I/O-Bound Tasks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Unbounded (1000 goroutines) | 3,689,054 | **201,160** | **3,010** |
| Pool 32 workers | 37,615,836 | **36,488** | **67** |
| Pool 64 workers | 20,049,604 | **40,175** | **132** |

**Trade-off visible:**
- Unbounded: fastest wall-clock, but 201 KB allocated + 3010 allocs + 1000 concurrent connections
- Pool 64: 5.4x slower wall-clock, but 80% less memory + 23x fewer allocs + max 64 connections

## 🔄 errgroup.SetLimit() — Idiomatic Alternative

Go's `golang.org/x/sync/errgroup` with `SetLimit()` provides bounded concurrency with less boilerplate. Here's how it compares to the custom worker pool:

### Implementation

```go
func ProcessWithErrgroup(tasks []Task, workers int) []int {
    results := make([]int, len(tasks))

    g, _ := errgroup.WithContext(context.Background())
    g.SetLimit(workers)

    for i, task := range tasks {
        i, task := i, task
        g.Go(func() error {
            results[i] = ProcessTask(task)
            return nil
        })
    }

    _ = g.Wait()
    return results
}
```

### Benchmark: Custom Pool vs errgroup

> Machine: Apple M1, Go 1.25.0

#### CPU-Bound (100 tasks, no I/O)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Custom Pool 8 workers | 692,800 | **3,813** | **11** |
| Custom Pool 16 workers | 401,491 | **3,912** | **19** |
| errgroup limit=8 | 385,720 | 9,970 | 205 |
| errgroup limit=16 | 480,778 | 9,971 | 205 |

#### I/O-Bound (100 tasks, 1ms I/O)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Custom Pool 32 workers | **6,954,025** | **7,259** | **67** |
| errgroup limit=32 | 11,088,427 | 19,594 | 305 |

#### Scale: 1000 I/O-Bound Tasks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Custom Pool 64 workers | **24,605,448** | **40,342** | **134** |
| errgroup limit=64 | 50,642,654 | 192,651 | 3,006 |

### Verdict: When to Use Which

| Criteria | Custom Worker Pool | errgroup.SetLimit() |
|----------|-------------------|---------------------|
| **Simplicity** | More boilerplate | ✅ 5 lines of code |
| **Memory efficiency** | ✅ 3-5x less allocations | Higher per-task overhead |
| **Error propagation** | Manual | ✅ Built-in (first error cancels) |
| **Backpressure** | ✅ Channel blocks when full | No backpressure signal |
| **Long-lived pools** | ✅ Start once, submit many | New group per batch |
| **Reusable workers** | ✅ Workers persist | Goroutine per task |
| **Performance at scale** | ✅ Better ns/op and allocs | Acceptable for most cases |

### Recommendation

**Use `errgroup.SetLimit()` when:**
- Processing a batch of independent tasks (most common case)
- You want error propagation from any task
- Code simplicity matters more than raw performance
- Task count is moderate (< 10K)

**Use custom worker pool when:**
- You need backpressure (block submitters when pool is busy)
- Workers are long-lived and process continuous streams
- Memory efficiency is critical (high-throughput, millions of tasks)
- You need worker-local state or connection reuse

## ⚡ Implementation

### Channel-Based Worker Pool

```go
type WorkerPool struct {
    workers int
    jobs    chan Task
    wg      sync.WaitGroup
}

func NewWorkerPool(workers, bufferSize int) *WorkerPool {
    return &WorkerPool{
        workers: workers,
        jobs:    make(chan Task, bufferSize),
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for job := range p.jobs {
                ProcessTask(job)
            }
        }()
    }
}

func (p *WorkerPool) Submit(task Task) {
    p.jobs <- task  // Blocks if buffer full = backpressure
}

func (p *WorkerPool) Wait() {
    close(p.jobs)
    p.wg.Wait()
}
```

### Key Design Decisions

1. **Buffered channel** — allows submitters to continue without blocking immediately
2. **Range over channel** — workers exit cleanly when channel is closed
3. **WaitGroup** — caller knows when all work is done
4. **Backpressure** — when buffer is full, `Submit` blocks (prevents overload)

## 💰 Cost Impact

### Memory: Unbounded vs Pool

```
1M tasks, unbounded:
  Goroutine stacks: 1,000,000 × 4KB = 4 GB
  Allocations: ~3M objects → heavy GC pressure
  Peak memory: 4-8 GB

1M tasks, pool(64):
  Goroutine stacks: 64 × 4KB = 256 KB
  Allocations: ~130 objects → minimal GC
  Peak memory: ~50 MB (task data only)

Savings: 99.9% less goroutine memory
```

### Upstream Protection

```
Scenario: Process 10K orders, each needs DB query

Unbounded:
  10,000 concurrent DB connections
  PostgreSQL max_connections = 100 → 9,900 FAIL
  Error rate: 99%

Pool(50):
  50 concurrent DB connections
  All within PostgreSQL limit
  Error rate: 0%
  Throughput: 50 queries × 1000/sec = 50K queries/sec
```

### AWS Cost

```
Unbounded (needs large instance for memory):
  Instance: r5.xlarge (32GB RAM) = $200/month

Pool (controlled memory):
  Instance: t3.medium (4GB RAM) = $30/month

Savings: $170/month per service
```

## 🧪 How to Run

```bash
cd patterns/worker-pool

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Longer benchmark
go test -bench=. -benchmem -benchtime=3s
```

## 📚 Key Takeaways

1. **Use pools for batch processing** — never spawn 1 goroutine per item at scale
2. **Pool size = target concurrency to upstream** — match your DB/API connection limits
3. **Unbounded is OK for small, fast tasks** — 100 goroutines is fine, 100K is not
4. **Buffered channels provide backpressure** — system won't accept more than it can handle
5. **CPU-bound: workers = runtime.NumCPU()** — more workers won't help
6. **I/O-bound: workers = upstream capacity** — match DB pool size or API rate limit

### Pool Sizing Guide

| Workload Type | Recommended Workers | Reasoning |
|---------------|--------------------:|-----------|
| CPU-bound | `runtime.NumCPU()` | More workers = context switching waste |
| I/O-bound (DB) | DB `max_connections / 2` | Leave room for other services |
| I/O-bound (HTTP) | 20-100 | Match upstream rate limits |
| Mixed | `NumCPU() * 2` to `NumCPU() * 4` | Balance CPU and I/O overlap |

### When to Use What

| Scenario | Pattern |
|----------|---------|
| <100 tasks, fast | Unbounded goroutines (simple) |
| >100 tasks, independent, one-shot | `errgroup.SetLimit()` (idiomatic) |
| >100 tasks, need backpressure | Custom worker pool |
| Batch processing (1K+) | Worker pool or errgroup |
| Rate-limited upstream | Worker pool (workers = rate limit) |
| Long-lived stream processing | Worker pool (reusable workers) |
| Need error from first failure | `errgroup` (built-in cancellation) |


## When This Is Acceptable

- Spawning a small, bounded number of goroutines (< 10) for parallel I/O
- Short-lived goroutines in request handlers that are bounded by request concurrency
- Fan-out patterns where the number of goroutines equals the number of known tasks
