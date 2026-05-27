# Error Handling Efficiency

## 📋 Overview

Reducing heap allocations from error creation on hot paths using sentinel errors and pre-allocated custom error types.

## TL;DR
- **Problem**: `errors.New()` and `fmt.Errorf()` allocate on every call; on hot paths this creates millions of unnecessary heap allocations
- **Solution**: Sentinel errors (`var ErrX = errors.New(...)`) and pre-allocated custom error types — zero allocation on use
- **Impact**: 5% error rate × 100M req/day = 5M unnecessary allocations/day eliminated

## 🎯 Problem Statement

In Go, `errors.New()` allocates a new error object on the heap every time it's called. `fmt.Errorf()` is even worse — it allocates for format string processing and argument boxing. On hot paths with even moderate error rates, this creates millions of avoidable heap allocations per day.

**Real-world impact:** A service handling 100M requests/day with a 5% error rate generates 5M heap allocations/day just from error creation!

## 🔍 Root Cause

### Why error creation allocates:

1. **`errors.New()`**: Creates a new `*errorString` struct on the heap every call (~16 bytes)
2. **`fmt.Errorf()`**: Allocates for format processing + argument boxing (~64-80 bytes, 2-3 allocs)
3. **Interface boxing**: Returning `error` interface requires heap allocation of the underlying value

### The allocation chain:

```go
// ❌ BAD: Every call allocates a new error on the heap
func processRequest() error {
    if notFound {
        return errors.New("resource not found")  // 1 alloc, 16 bytes
    }
    if invalid {
        return fmt.Errorf("invalid resource %s at %d", name, id)  // 2-3 allocs, 64-80 bytes
    }
    return nil
}
```

At 5% error rate on a hot path:
- 100M requests/day × 5% = **5M allocations/day**
- Each allocation adds GC tracking overhead (~8 bytes metadata)
- More allocations → more frequent GC cycles → higher p99 latency

## ⚡ Solution

Use sentinel errors (package-level variables) and pre-allocated custom error types. These are allocated once at program init and reused with zero allocation on every subsequent use.

### Sentinel Errors (Zero Allocation)

```go
// ✅ GOOD: Allocated once at init, zero alloc on every use
var (
    ErrNotFound     = errors.New("resource not found")
    ErrUnauthorized = errors.New("unauthorized access")
    ErrRateLimit    = errors.New("rate limit exceeded")
)

func processRequest() error {
    if notFound {
        return ErrNotFound  // Zero allocation — just a pointer copy
    }
    return nil
}
```

### Pre-allocated Custom Error Types (Zero Allocation + Rich Context)

```go
// ✅ GOOD: Custom error type with pre-built message
type AppError struct {
    Code    int
    Message string
}

func (e *AppError) Error() string {
    return e.Message
}

// Allocated once, reused forever
var (
    ErrNotFound = &AppError{Code: 404, Message: "resource not found"}
    ErrConflict = &AppError{Code: 409, Message: "resource conflict"}
)
```

### When You Need Dynamic Context

```go
// For cases where you need dynamic error info on non-hot paths:
// Use fmt.Errorf() — the allocation cost is acceptable off hot paths
func processRareError(id string) error {
    return fmt.Errorf("unexpected state for resource %s: manual investigation needed", id)
}
```

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

| **Method** | **ns/op** | **B/op** | **allocs/op** |
| --- | --- | --- | --- |
| `errors.New()` | ~25 ns | 16 B | 1 |
| `fmt.Errorf()` | ~120 ns | 64-80 B | 2-3 |
| Sentinel error | ~0.3 ns | 0 B | 0 |
| Pre-allocated custom | ~0.3 ns | 0 B | 0 |

### Hot Path Throughput (5% error rate, 1M iterations)

| **Method** | **Relative Speed** | **Total Allocs** |
| --- | --- | --- |
| `errors.New()` | 1x (baseline) | ~50,000 |
| `fmt.Errorf()` | ~0.5x (slower) | ~100,000-150,000 |
| Sentinel error | ~1.5x (faster) | 0 |
| Pre-allocated custom | ~1.5x (faster) | 0 |

```bash
# Run benchmarks yourself
cd patterns/error-handling
go test -bench=. -benchmem -benchtime=3s
```

## 💰 Cost Impact

### Daily Allocation Waste (100M req/day, 5% error rate = 5M errors/day)

| **Method** | **Allocs/day** | **Bytes/day** | **GC Overhead** |
| --- | --- | --- | --- |
| `errors.New()` | 5,000,000 | ~76 MB | ~38 MB tracking |
| `fmt.Errorf()` | 10,000,000+ | ~343 MB | ~76 MB tracking |
| Sentinel/Custom | 0 | 0 | 0 |

### Impact on Production

```
Switching from errors.New() to sentinel errors eliminates:
  • 5,000,000 heap allocations/day
  • ~76 MB/day of allocation pressure
  • Reduced GC frequency → lower p99 latency

Switching from fmt.Errorf() to sentinel errors eliminates:
  • 10,000,000+ heap allocations/day
  • ~343 MB/day of allocation pressure
  • Significantly reduced GC pauses
```

### AWS Cost Projection

At scale, fewer allocations → less GC pressure → lower p99 latency → smaller instances needed:

| **Scale** | **Allocs Eliminated** | **Memory Pressure Saved** |
| --- | --- | --- |
| 10M req/day | 500K/day | ~7.6 MB/day |
| 100M req/day | 5M/day | ~76 MB/day |
| 1B req/day | 50M/day | ~760 MB/day |

The real savings come from reduced GC CPU time and lower tail latency, which can defer instance scaling.

## When to Apply

✅ **DO apply when:**

- Error occurs on a hot path (high-frequency operations)
- Error rate is non-trivial (≥1% of requests)
- Service handles high throughput (1M+ req/day)
- You're seeing GC pressure in production profiles
- The error message is static/predictable

❌ **DON'T over-optimize when:**

- Error path is rarely hit (< 0.01% of requests)
- You need dynamic context in the error message (use `fmt.Errorf` on cold paths)
- The function is called infrequently
- Readability would suffer significantly

### Decision Guide

```
Is this a hot path? (>10K calls/sec)
├── YES → Is the error message static?
│         ├── YES → Use sentinel error (var ErrX = errors.New(...))
│         └── NO  → Use pre-allocated custom error type
└── NO  → Use errors.New() or fmt.Errorf() freely
```

## 🧪 How to Run

```bash
cd patterns/error-handling

# Run demo (shows allocation counts and cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Check escape analysis (see what allocates)
go build -gcflags="-m" .
```
