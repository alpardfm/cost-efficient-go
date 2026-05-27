# Connection Pooling

## 📋 Overview

Reusing TCP connections instead of creating new ones per request — eliminating handshake overhead and reducing latency by 2.7x.

## TL;DR
- **Problem**: Creating a new TCP connection per request adds 0.5-5ms handshake overhead and wastes file descriptors
- **Solution**: Reuse connections via a pool (`database/sql`, `http.Transport`, or custom pool with configurable maxIdle)
- **Impact**: 2.7x faster latency, 40x less memory, 16x fewer allocations; can downgrade DB instance saving ~$1,150/month

## 🎯 Problem Statement

Every new TCP connection requires:
1. **TCP 3-way handshake** — SYN → SYN-ACK → ACK (~1 RTT)
2. **TLS negotiation** — additional 1-2 RTTs for HTTPS
3. **OS resource allocation** — file descriptors, kernel buffers

For a database or external service call, this adds **0.5-5ms per request** depending on network distance.

**Real-world impact:** At 1000 req/sec, connection-per-request wastes 0.5-5 seconds of pure handshake time every second.

## 🔍 Root Cause Analysis

### Why connection-per-request is expensive:

```
Request 1: [dial 0.5ms] [write] [read] [close]
Request 2: [dial 0.5ms] [write] [read] [close]
Request 3: [dial 0.5ms] [write] [read] [close]
...
```

### Why pooling is fast:

```
Request 1: [dial 0.5ms] [write] [read] [return to pool]
Request 2: [get from pool] [write] [read] [return to pool]  ← no dial!
Request 3: [get from pool] [write] [read] [return to pool]  ← no dial!
...
```

After the first request, subsequent requests skip the dial entirely.

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4 (localhost TCP echo server):

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Connection per request | **560,731** | 2,779 | **32** |
| Pooled connection | **207,325** | 69 | **2** |
| Pooled concurrent (8 goroutines) | **53,009** | 69 | **2** |

### Key Results

| Metric | Per-Request | Pooled | Improvement |
|--------|-------------|--------|-------------|
| Latency | 561 µs | 207 µs | **2.7x faster** |
| Memory | 2,779 B | 69 B | **40x less** |
| Allocations | 32 | 2 | **16x fewer** |
| Concurrent throughput | - | 53 µs | **10.6x faster** |

### Pool Size Impact (100 sequential requests)

| maxIdle | Connections Created | Reused | Reuse Ratio |
|---------|--------------------:|-------:|------------:|
| 1 | 1 | 99 | 99% |
| 5 | 1 | 99 | 99% |
| 10 | 1 | 99 | 99% |
| 20 | 1 | 99 | 99% |

**For sequential workloads, even maxIdle=1 achieves 99% reuse.** Pool size matters for concurrent workloads.

### Pool Size Impact Under Concurrency

When multiple goroutines compete for connections simultaneously, pool size becomes critical.
With an undersized pool, connections are discarded on return (pool full) and must be re-created on next request.

**Benchmark: concurrency=50, varying pool sizes** (Apple M1, Go 1.24.4):

| Pool Size (maxIdle) | ns/op | B/op | Notes |
|--------------------:|------:|-----:|-------|
| 1 | **76,564** | 144 | Pool too small — constant dial overhead |
| 5 | 50,499 | 125 | Still undersized for 50 goroutines |
| 10 | 48,571 | 113 | Improving but not enough |
| 25 | **37,608** | 100 | Sweet spot — pool ≈ concurrency/2 |
| 50 | 48,243 | 95 | Matched — diminishing returns |
| 100 | 48,243 | 95 | Oversized — no additional benefit |

**Key insight:** `poolSize=1` with `concurrency=50` is **2x slower** than `poolSize=25`.

### Pool Size Mismatch Benchmark

Direct comparison showing undersized vs matched pool at concurrency=50:

| Scenario | ns/op | B/op | allocs/op |
|----------|------:|-----:|----------:|
| Undersized (pool=5, conc=50) | **43,375** | 92 | 2 |
| Matched (pool=50, conc=50) | **25,362** | 81 | 2 |

**Matched pool is 1.7x faster** — the undersized pool forces TCP dials when all 5 idle slots are occupied.

### Why Pool Size Matters for Concurrent Workloads

```
Sequential (concurrency=1):
  Get → [use] → Put → Get → [use] → Put
  Pool always has 1 idle connection ready. maxIdle=1 is fine.

Concurrent (concurrency=50):
  Goroutine 1: Get → [use 500µs] → Put
  Goroutine 2: Get → [use 500µs] → Put
  ...
  Goroutine 50: Get → pool empty! → dial new connection (expensive)

  With maxIdle=5: only 5 connections survive between rounds
  With maxIdle=50: all 50 connections survive → zero dials after warmup
```

## ⚡ Implementation

### Simple Pool Pattern

```go
type Pool struct {
    mu      sync.Mutex
    conns   []net.Conn
    factory func() (net.Conn, error)
    maxIdle int
}

func (p *Pool) Get() (net.Conn, error) {
    p.mu.Lock()
    if len(p.conns) > 0 {
        conn := p.conns[len(p.conns)-1]
        p.conns = p.conns[:len(p.conns)-1]
        p.mu.Unlock()
        return conn, nil  // Reuse existing connection
    }
    p.mu.Unlock()
    return p.factory()    // Create new only when pool empty
}

func (p *Pool) Put(conn net.Conn) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if len(p.conns) >= p.maxIdle {
        conn.Close()      // Pool full, discard
        return
    }
    p.conns = append(p.conns, conn)
}
```

### Pool Sizing Guidelines

| Workload | Recommended maxIdle | Reasoning |
|----------|--------------------:|-----------|
| Low traffic (<100 req/s) | 5 | Minimal connections, low overhead |
| Medium traffic (100-1K req/s) | 10-20 | Balance between reuse and resource usage |
| High traffic (>1K req/s) | 20-50 | Enough for concurrent burst |
| Burst-heavy | 2x avg concurrency | Handle spikes without creating new connections |

### Go stdlib: `database/sql` Pool

Go's `database/sql` already implements connection pooling:

```go
db, _ := sql.Open("postgres", connStr)
db.SetMaxOpenConns(25)          // Max total connections
db.SetMaxIdleConns(10)          // Max idle connections in pool
db.SetConnMaxLifetime(5*time.Minute)  // Recycle stale connections
db.SetConnMaxIdleTime(1*time.Minute)  // Close idle connections after 1min
```

### Go stdlib: `http.Transport` Pool

HTTP client also pools by default:

```go
transport := &http.Transport{
    MaxIdleConns:        100,              // Total idle across all hosts
    MaxIdleConnsPerHost: 10,               // Per-host idle limit
    IdleConnTimeout:     90 * time.Second, // Close idle after 90s
}
client := &http.Client{Transport: transport}
```

## 💰 Cost Impact

### Per-Request Savings

```
Without pool: 561 µs per request (dial + write + read + close)
With pool:    207 µs per request (get + write + read + return)
Saved:        354 µs per request (63% reduction)
```

### At Scale: 1000 req/sec to PostgreSQL

```
Without pool:
  Connection time: 561 µs × 1000 = 561 ms/sec of dial overhead
  File descriptors: 1000 simultaneous (may hit OS limit)
  PostgreSQL processes: 1000 forked (each ~5-10MB RAM)
  DB RAM overhead: 5-10 GB just for connections

With pool (maxIdle=25):
  Connection time: ~0 ms/sec (all reused)
  File descriptors: 25 persistent
  PostgreSQL processes: 25 persistent (~125-250MB RAM)
  DB RAM overhead: 125-250 MB

Monthly savings (AWS RDS):
  Without pool: db.r5.2xlarge needed ($1,200/month) for connection RAM
  With pool: db.t3.medium sufficient ($50/month)
  Savings: ~$1,150/month
```

### Real-World: Connection Exhaustion

```
Scenario: 500 concurrent requests, no pool, PostgreSQL max_connections=100

Result: 400 requests FAIL with "too many connections"
Error rate: 80%

With pool (maxOpen=50):
Result: All 500 requests succeed (queued, not rejected)
Error rate: 0%
```

## 🧪 How to Run

```bash
cd patterns/connection-pooling

# Run demo (starts local echo server, compares pooled vs non-pooled)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Longer benchmark
go test -bench=. -benchmem -benchtime=5s

# Run only concurrent pool size benchmarks (shows when pool size matters)
go test -bench=BenchmarkConcurrentPoolSize -benchmem

# Run pool size mismatch comparison
go test -bench=BenchmarkPoolSizeMismatch -benchmem

# Run pool efficiency test (shows created vs reused stats)
go test -v -run TestConcurrentPoolEfficiency
```

## 📚 Key Takeaways

1. **Always pool database connections** — Go's `database/sql` does this automatically, just configure the limits
2. **Pool HTTP client connections** — use a shared `http.Client` with configured `Transport`, never create per-request
3. **Pool size = expected concurrency** — not total requests, but simultaneous active connections
4. **Set idle timeouts** — stale connections waste resources and may be closed by the server
5. **Monitor pool stats** — `db.Stats()` shows open, idle, and wait counts

### When to Apply

✅ **DO pool when:**
- Connecting to databases (PostgreSQL, MySQL, Redis)
- Calling external HTTP APIs
- Using message queues (RabbitMQ, Kafka)
- Any TCP-based service called repeatedly

❌ **DON'T pool when:**
- One-shot CLI tools (connect once, do work, exit)
- WebSocket connections (already long-lived)
- Connection is used once then never again
