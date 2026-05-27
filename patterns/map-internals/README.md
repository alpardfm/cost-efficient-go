# Map Internals & Memory Overhead

## 📋 Overview

Understanding Go map's hidden memory costs and when to use alternatives for better performance.

## TL;DR
- **Problem**: Go maps use ~24 bytes overhead per entry (total ~48 bytes for map[int]string), 3x more memory than equivalent slices
- **Solution**: Pre-allocate maps with `make(map[K]V, size)`, use slices for sequential keys, use `map[T]struct{}` for sets
- **Impact**: 4.3x faster insertion, 3.6x faster iteration, 67% memory reduction vs naive maps

## 🎯 Problem Statement

**Go maps use 3-10x more memory than equivalent slices!** Each map entry uses ~48 bytes total (key + value + ~24 bytes overhead for hash table metadata).

## 🔍 Root Cause

### Map Internals (Hash Table):

Each map entry contains:
- Key data (8 bytes for int)
- Value data (16 bytes for string)
- Overhead (~24 bytes): tophash, bucket padding, amortized overflow pointer

Total: ~48 bytes per entry (24 bytes data + ~24 bytes overhead)

### Where Does the ~24 Bytes Overhead Come From?

1. **Tophash array** (1 byte per slot × 8 slots per bucket, amortized ~1 byte/entry)
2. **Bucket alignment padding** (struct alignment to 8-byte boundaries)
3. **Overflow pointer** (8 bytes per bucket, amortized across 8 slots)
4. **Load factor waste** (buckets only ~80% full on average, wasting ~20% capacity)

### The Hidden Rehashing Problem:

```go
// Map growth causes REHASHING of ALL entries!
m := make(map[int]string)      // 1 bucket
// When load factor > 6.5: DOUBLE buckets, REHASH EVERY ENTRY
```

## ⚡ Solution

### 1. Pre-allocate Maps

```go
// ❌ BAD: Causes multiple rehashes
m := make(map[int]string)
for i := 0; i < 1000; i++ {
    m[i] = "value"  // Rehashes multiple times!
}

// ✅ GOOD: Single allocation, no rehashes
m := make(map[int]string, 1000)
for i := 0; i < 1000; i++ {
    m[i] = "value"  // No rehashes!
}
```

### 2. Use map[T]struct{} for Sets

```go
// ❌ BAD: Wasteful
seen := make(map[string]bool)

// ✅ GOOD: Zero-byte values
seen := make(map[string]struct{})
seen["item"] = struct{}{}
```

### 3. Choose the Right Data Structure

```go
// Use MAP when: O(1) lookup critical, keys are sparse/random
// Use SLICE when: iteration frequent, keys are sequential integers, memory constrained
```

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

```text
Insertion:
Benchmark_MapInsert_1000-8          50000     24567 ns/op   50240 B/op     11 allocs/op
Benchmark_MapInsertPrealloc_1000-8 200000      8923 ns/op   16480 B/op      1 allocs/op
Benchmark_SliceStructInsert_1000-8 300000      5678 ns/op   16384 B/op      1 allocs/op

Lookup:
Benchmark_MapLookup-8              5000000       256 ns/op       0 B/op      0 allocs/op
Benchmark_SliceLookupDirect-8     20000000        75 ns/op       0 B/op      0 allocs/op

Iteration:
Benchmark_MapIteration-8             50000     32415 ns/op       0 B/op      0 allocs/op
Benchmark_SliceIteration-8          200000      8923 ns/op       0 B/op      0 allocs/op
```

| **Operation** | **Improvement** | **Why** |
| --- | --- | --- |
| Insertion | 4.3x faster | No rehashing, fewer allocations |
| Lookup | 3.4x faster (slice) | Cache locality |
| Iteration | 3.6x faster | Sequential memory access |
| Memory | 67% reduction | No per-entry overhead |

## 💰 Cost Impact

### Scenario: 1M user ID → name mappings

```text
Map[int]string:      47.68 MB  (~48 bytes/entry × 1M)
Slice of structs:    15.26 MB  (16 bytes/entry × 1M)
Map overhead:        32.42 MB  (3.1x more!)
```

**Monthly Cost:**

```text
Map cost:            $0.179/month
Slice cost:          $0.057/month
Monthly savings:     $0.122/month
```

**Scaling Impact:**

| **Entries** | **Annual Savings** |
| --- | --- |
| 1M | $1.46 |
| 10M | $14.64 |
| 100M | $146.40 |
| 1B | $1,464.00 |

## When to Apply

### Use Maps when:
- Sparse data (many missing keys)
- O(1) lookup critical
- Set operations (union, intersection)
- Non-sequential keys

### Use Slices when:
- Sequential integer keys (0,1,2,3...)
- Frequent iteration
- Memory-constrained environments
- Need cache locality

## 🧪 How to Run

```bash
cd patterns/map-internals

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Run all benchmarks with longer duration
go test -bench=. -benchmem -benchtime=2s
```
