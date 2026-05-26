# Profiling & Benchmarking Techniques

## 📋 Overview

How to measure Go performance correctly — avoiding common pitfalls like compiler elimination, cold-start bias, and misleading averages.

## 🎯 Problem Statement

Most developers benchmark incorrectly:

- **Compiler eliminates dead code** — if the result isn't used, the compiler may skip the computation entirely
- **Averages hide tail latency** — P50 might be 1ms but P99 could be 50ms
- **Cold-start bias** — first iterations are slower due to cache misses and JIT-like effects
- **Wrong scale** — benchmarking 100 items doesn't predict behavior at 100K items

**Real-world impact:** Bad benchmarks lead to wrong optimization decisions — you optimize the wrong thing or miss the real bottleneck.

## 🔍 Key Techniques

### 1. Prevent Compiler Elimination

```go
// ❌ BAD: Compiler may optimize away the entire call
func BenchmarkSort(b *testing.B) {
    for i := 0; i < b.N; i++ {
        SortInts(data) // Result unused!
    }
}

// ✅ GOOD: Package-level sink prevents elimination
var sink interface{}

func BenchmarkSort(b *testing.B) {
    for i := 0; i < b.N; i++ {
        sink = SortInts(data) // Compiler can't eliminate
    }
}
```

### 2. Percentile-Based Timing

```go
// Don't just measure average — measure P50, P95, P99
func MeasureTime(fn func(), iterations int) (p50, p95, p99 time.Duration) {
    durations := make([]time.Duration, iterations)
    
    // Warm-up: discard first 10%
    for i := 0; i < iterations/10; i++ {
        fn()
    }
    
    // Measure
    for i := 0; i < iterations; i++ {
        start := time.Now()
        fn()
        durations[i] = time.Since(start)
    }
    
    sort.Slice(durations, func(i, j int) bool {
        return durations[i] < durations[j]
    })
    
    return durations[len(durations)*50/100],
           durations[len(durations)*95/100],
           durations[len(durations)*99/100]
}
```

### 3. Memory Measurement

```go
func MeasureAllocs(fn func()) uint64 {
    runtime.GC() // Clean slate
    var before, after runtime.MemStats
    runtime.ReadMemStats(&before)
    fn()
    runtime.ReadMemStats(&after)
    return after.TotalAlloc - before.TotalAlloc
}
```

### 4. Scale Testing

Always benchmark at multiple scales to understand algorithmic complexity:

```go
BenchmarkSort100    // O(n log n) at small n
BenchmarkSort10K    // Where does it start to hurt?
BenchmarkSort100K   // Production-realistic scale
```

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4:

### Compiler Elimination Impact

| Benchmark | ns/op | Note |
|-----------|-------|------|
| SortBad (result unused) | 2,255,041 | Compiler may partially optimize |
| SortGood (sink used) | 1,838,662 | Accurate measurement |

**Note:** In this case the "bad" version appears slower because the compiler's optimization is inconsistent — sometimes it partially eliminates, sometimes not. The point: always use a sink for reliable results.

### Allocation Scaling

| Size per slice | ns/op | B/op | allocs/op |
|----------------|-------|------|-----------|
| 1 KB × 100 | 64,124 | 105,112 | 102 |
| 4 KB × 100 | 264,942 | 412,314 | 102 |
| 64 KB × 100 | 1,468,311 | 6,556,318 | 102 |

**Key insight:** Same number of allocations (102), but 4x size = 4x time. Memory bandwidth is the bottleneck, not allocation count.

### Processing Scale (CPU-bound)

| Scale | ns/op | Ratio |
|-------|-------|-------|
| 100 items | 262 | 1x |
| 10K items | 15,440 | 59x |
| 1M items | 1,368,472 | 5,224x |

**Linear scaling confirmed** — processing 10,000x more items takes ~5,000x longer (expected for O(n) work).

### Sort Scale (O(n log n))

| Scale | ns/op | B/op | Ratio |
|-------|-------|------|-------|
| 100 items | 3,388 | 920 | 1x |
| 10K items | 1,552,389 | 81,944 | 458x |
| 100K items | 23,534,372 | 802,843 | 6,947x |

**Super-linear scaling** — 1000x more items = 6,947x slower. This is O(n log n) in action. At 100K items, a single sort takes 23ms — enough to impact API latency.

## 💰 Cost Impact

### Why Correct Benchmarking Saves Money

| Scenario | Bad Benchmark | Correct Benchmark | Decision Impact |
|----------|---------------|-------------------|-----------------|
| Sort 10K items | "Fast enough" (wrong) | 1.5ms per call | Need caching for hot paths |
| Allocate 64KB × 100 | "Only 102 allocs" | 6.5MB allocated | Need object pooling |
| Batch 1M items | "Linear, no problem" | 1.4ms per batch | Acceptable for background jobs |

### Real Example: API Response Sorting

```
Scenario: Sort 10K items per API response, 1000 req/sec

CPU time per second: 1.55ms × 1000 = 1.55 seconds of sort time
→ Need >2 CPU cores just for sorting!

With caching (sort once, serve many):
CPU time per second: 1.55ms × 1 (cache hit ratio 99.9%) = 0.0015 seconds
→ 1000x reduction in CPU cost
```

## 🧪 How to Run

```bash
cd patterns/profiling-benchmarking

# Run demo (shows percentile timing, memory measurement, GC stats)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Generate CPU profile
go test -bench=BenchmarkSort10K -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Generate memory profile
go test -bench=BenchmarkAllocate4KB -memprofile=mem.prof
go tool pprof -http=:8081 mem.prof

# Compare two implementations
go test -bench=. -count=5 > old.txt
# ... make changes ...
go test -bench=. -count=5 > new.txt
benchstat old.txt new.txt
```

## 📚 Key Takeaways

1. **Always use a sink** — `var sink interface{}` prevents compiler elimination
2. **Measure percentiles, not averages** — P99 reveals real user experience
3. **Warm up before measuring** — discard first 10% of iterations
4. **Test at production scale** — 100 items ≠ 100K items
5. **Use `b.ReportAllocs()`** — allocation count matters as much as speed
6. **Profile before optimizing** — `go tool pprof` shows WHERE time is spent

### Essential Tools

| Tool | Purpose |
|------|---------|
| `go test -bench` | Micro-benchmarks |
| `go test -cpuprofile` | CPU profiling |
| `go test -memprofile` | Memory profiling |
| `go tool pprof` | Profile visualization |
| `benchstat` | Statistical comparison |
| `runtime.MemStats` | Runtime memory inspection |
| `runtime/trace` | Execution tracing (goroutines, GC) |

### When to Profile

✅ **DO profile when:**
- API latency exceeds SLA
- Memory usage grows unexpectedly
- GC pauses cause latency spikes
- You need to choose between two implementations

❌ **DON'T profile when:**
- Code is correct but "feels slow" without data
- Optimizing code that runs once at startup
- The bottleneck is I/O (network, disk), not CPU/memory
