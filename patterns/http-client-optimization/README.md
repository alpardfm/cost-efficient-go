# HTTP Client Optimization

## 📋 Overview

Configuring Go's HTTP client for production — proper timeouts, connection reuse, body handling, and context cancellation.

## TL;DR
- **Problem**: Default `http.Client` has no timeout, leaks connections when body isn't drained, and creates new clients per request
- **Solution**: Share one client with configured Transport, always drain+close body, use context for cancellation
- **Impact**: 2.6x faster with body drain, 18% faster with shared client, prevents goroutine leaks and OOM crashes

## 🎯 Problem Statement

Go's default `http.Client` is not production-ready:

- **No timeout** — request hangs forever if upstream is slow
- **Body not drained** — connection can't be reused (leaked to pool)
- **New client per request** — no connection reuse across calls
- **No context** — can't cancel requests when caller gives up

**Real-world impact:** A single slow upstream without timeout can exhaust all goroutines and crash your service.

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4 (localhost httptest server):

### Body Handling Impact

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Bad (no body close) | **1,558,876** | **28,188** | **137** |
| Good (body drained) | **596,442** | **6,100** | **73** |

**Properly closing body: 2.6x faster, 4.6x less memory, 1.9x fewer allocations.**

Why? Without draining the body, the TCP connection can't be returned to the pool. Each request creates a new TCP connection (handshake overhead).

### Shared Client vs New Client Per Request

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| New client per request | 351,577 | 6,165 | 74 |
| Shared client | **287,379** | **6,110** | **73** |

**Shared client: 18% faster** — transport pool is reused.

### Sequential vs Concurrent (10 requests)

| Benchmark | ns/op | Throughput |
|-----------|-------|-----------|
| Sequential (10 requests) | 3,973,147 | 2.5 req/ms |
| Concurrent (10 requests) | **1,632,352** | **6.1 req/ms** |

**Concurrent: 2.4x faster** — requests overlap I/O wait time.

## ⚡ Production Configuration

### The Correct Client Setup

```go
client := &http.Client{
    Timeout: 10 * time.Second,  // Total request timeout
    Transport: &http.Transport{
        MaxIdleConns:        100,              // Total idle pool
        MaxIdleConnsPerHost: 20,               // Per-host idle
        MaxConnsPerHost:     100,              // Per-host max
        IdleConnTimeout:     90 * time.Second, // Close stale
        DisableKeepAlives:   false,            // Reuse connections
    },
}
```

### The Correct Request Pattern

```go
func DoRequest(ctx context.Context, client *http.Client, url string) ([]byte, error) {
    // 1. Use context for cancellation
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }

    // 2. Execute request
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()  // 3. ALWAYS close body

    // 4. Read body (enables connection reuse)
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return body, nil
}
```

### Common Mistakes

```go
// ❌ BAD: No timeout — hangs forever
client := &http.Client{}

// ❌ BAD: Body not closed — connection leaked
resp, _ := client.Get(url)
return resp.StatusCode  // body never read!

// ❌ BAD: New client per request — no pool reuse
func handler(w http.ResponseWriter, r *http.Request) {
    client := &http.Client{Timeout: 5*time.Second}  // Created every request!
    client.Get(url)
}

// ❌ BAD: No context — can't cancel
resp, _ := client.Get(url)  // If upstream takes 60s, you wait 60s
```

## 💰 Cost Impact

### Connection Leak (no body drain)

```
Without body drain:
  Per request: 1,559 µs + new TCP connection each time
  At 1000 req/sec: 1000 TCP handshakes/sec
  File descriptors: grows unbounded → eventual crash
  
With body drain:
  Per request: 596 µs (connection reused)
  At 1000 req/sec: ~20 persistent connections (pooled)
  File descriptors: stable at MaxIdleConnsPerHost
  
Savings: 2.6x less CPU, prevents OOM/fd exhaustion
```

### Timeout Protection

```
Scenario: Upstream goes down, no timeout configured

Without timeout:
  Goroutines pile up waiting forever
  Memory grows: 10KB/goroutine × 10,000 stuck = 100MB leaked
  Eventually: OOM kill → service restart → downtime

With timeout (10s):
  Goroutines released after 10s
  Error returned to caller
  Circuit breaker can trigger
  Service stays healthy

Cost of 1 hour downtime: $100-10,000+ depending on business
Cost of adding timeout: 2 lines of code
```

### Concurrent Requests

```
Sequential (calling 5 microservices):
  Total latency: 10ms + 15ms + 8ms + 12ms + 20ms = 65ms

Concurrent (same 5 services):
  Total latency: max(10, 15, 8, 12, 20) = 20ms

Savings: 69% latency reduction for fan-out patterns
```

## 🧪 How to Run

```bash
cd patterns/http-client-optimization

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Longer benchmark
go test -bench=. -benchmem -benchtime=3s
```

## 📚 Key Takeaways

1. **Always set a timeout** — `&http.Client{Timeout: 10*time.Second}`
2. **Always close and drain response body** — `defer resp.Body.Close()` + read/discard
3. **Share one client across requests** — create once, reuse everywhere
4. **Use context for cancellation** — `http.NewRequestWithContext(ctx, ...)`
5. **Fan-out concurrently** — don't call services sequentially if they're independent
6. **Configure transport** — tune `MaxIdleConnsPerHost` for your traffic pattern

### Transport Sizing Guide

| Traffic | MaxIdleConnsPerHost | MaxConnsPerHost |
|---------|--------------------:|----------------:|
| Low (<100 req/s per host) | 5 | 20 |
| Medium (100-1K req/s) | 20 | 100 |
| High (>1K req/s) | 50 | 200 |


## When This Is Acceptable

- Test code where client lifecycle matches test lifecycle
- CLI tools making a single HTTP request and exiting
- Clients that need unique TLS configurations per request (rare)
