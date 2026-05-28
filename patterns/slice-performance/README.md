# Slice Performance & Pre-allocation

## 📋 Overview

Optimizing slice usage in Go by understanding growth patterns and using pre-allocation to reduce allocations and improve performance.

## TL;DR
- **Problem**: Dynamic slice growth causes 11 reallocations and O(n) element copying for 1000 appends
- **Solution**: Pre-allocate with `make([]T, 0, capacity)` when size is known
- **Impact**: 3.8x faster, 10x fewer allocations, 0% memory waste vs ~50% waste

## 🎯 Problem Statement

Go slices grow dynamically by reallocating and copying data when capacity is exceeded. This causes:
- **Multiple memory allocations** (expensive syscalls)
- **Data copying** (O(n) operations)
- **Memory fragmentation**
- **Increased GC pressure**

## 🔍 Root Cause

### How Slices Work Internally:

```go
type sliceHeader struct {
    Data uintptr  // Pointer to underlying array
    Len  int      // Current length
    Cap  int      // Current capacity
} // 24 bytes on 64-bit systems
```

### Growth Algorithm:

1. If cap < 1024: double capacity
2. If cap ≥ 1024: grow by 25%
3. Each growth requires: allocate new array → copy all elements → GC old array

### The Cost of Naive Appends:

```text
Appending 1000 elements without pre-allocation:
Reallocations: 11 times
Copied elements: 1+2+4+8+16+32+64+128+256+512 = 1023 elements
Memory waste: ~50% on average
```

## ⚡ Solution

### Pre-allocate with `make()`

```go
// ❌ BAD: No pre-allocation
func processUsers(users []User) []UserResult {
    var results []UserResult  // Capacity = 0
    for _, user := range users {
        results = append(results, processUser(user))  // Causes reallocations!
    }
    return results
}

// ✅ GOOD: Pre-allocated
func processUsersOptimized(users []User) []UserResult {
    results := make([]UserResult, 0, len(users))
    for _, user := range users {
        results = append(results, processUser(user))  // No reallocations!
    }
    return results
}

// ✅ BETTER: When size is known upfront
func processUsersBest(users []User) []UserResult {
    results := make([]UserResult, len(users))
    for i, user := range users {
        results[i] = processUser(user)  // Direct assignment
    }
    return results
}
```

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

```text
BenchmarkNaiveAppend_1000-8     50000    24567 ns/op   40840 B/op   11 allocs/op
BenchmarkMakeAppend_1000-8     200000     8923 ns/op   16384 B/op    1 allocs/op
BenchmarkFixedArray_1000-8     300000     5678 ns/op   16384 B/op    1 allocs/op
```

| **Metric** | **Naive** | **Pre-allocated** | **Improvement** |
| --- | --- | --- | --- |
| Allocations | 11 | 1 | 91% reduction |
| Elements Copied | 1023 | 0 | 100% reduction |
| Memory Waste | ~50% | 0% | Perfect efficiency |
| Execution Time | 24,567 ns | 5,678 ns | 4.3x faster |

## 💰 Cost Impact

### Assumptions:
- 100 requests/second, each processes 1000 items
- AWS t3.medium: $0.0416/hour per vCPU

### Calculations:

```text
Time saved per request: 18,889 ns (77% faster)
Requests per day: 8,640,000
CPU seconds saved per day: 163.2 seconds
```

### Scaling Projections:

| **Request Rate** | **Annual Savings** |
| --- | --- |
| 100 RPS | $0.68 |
| 1,000 RPS | $6.79 |
| 10,000 RPS | $67.90 |
| 100,000 RPS | $679.00 |

Additional benefits: reduced GC pressure, better latency consistency, improved cache performance.

## When to Apply

✅ **DO apply when:**

- Processing slices in loops (database results, API responses)
- Building slices incrementally with known max size
- Working with slices > 100 elements
- In performance-critical code paths

❌ **DON'T over-optimize when:**

- Slices are tiny (< 10 elements)
- Code is not performance-critical
- Readability would suffer significantly

## 🧪 How to Run

```bash
cd patterns/slice-performance

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmarks (3 seconds each)
go test -bench=. -benchmem -benchtime=3s
```


## When This Is Acceptable

- Loops iterating fewer than 10 items where reallocation cost is negligible
- Slices that are immediately discarded after the loop (short-lived, GC-friendly)
- Cases where the final size is truly unknown and cannot be estimated
