# Memory Layout & Struct Alignment

## 📋 Overview

Optimizing Go struct memory usage by reducing padding through intelligent field ordering.

## 🎯 Problem Statement

Go aligns struct fields to natural memory boundaries (8 bytes on 64-bit systems). Poor field ordering can waste significant memory on padding bytes that contain no useful data.

**Real-world impact:** A struct with 4 fields can waste 50% of its memory on padding!

## 🔍 Root Cause Analysis

### Why padding happens:

1. **CPU Alignment:** CPUs read memory in word-sized chunks (8 bytes on 64-bit)
2. **Field Ordering:** Go compiler adds padding to align each field
3. **Wasted Space:** Padding bytes contain no useful data but consume memory

### Field Ordering Rules

For optimal memory layout, order fields from **largest to smallest** according to their natural alignment:

| Type | Size | Alignment | Priority |
| --- | --- | --- | --- |
| `[]T` (slice) | 24 bytes | 8 bytes | 1️⃣ Highest |
| `string` | 16 bytes | 8 bytes | 2️⃣ |
| `interface{}`, `any` | 16 bytes | 8 bytes | 2️⃣ |
| `int64`, `uint64`, `float64` | 8 bytes | 8 bytes | 3️⃣ |
| `*T` (pointer), `func()` | 8 bytes | 8 bytes | 3️⃣ |
| `int32`, `uint32`, `float32` | 4 bytes | 4 bytes | 4️⃣ |
| `int16`, `uint16` | 2 bytes | 2 bytes | 5️⃣ |
| `int8`, `uint8`, `bool`, `byte` | 1 byte | 1 byte | 6️⃣ Lowest |

### Common Anti-pattern:

```go
// ❌ BAD: Mixed field sizes cause maximum padding
type BadUser struct {
    ID     int32    // 4 bytes
    Active bool     // 1 byte → +3 bytes padding
    Name   string   // 16 bytes
    Age    int8     // 1 byte → +7 bytes padding
} // Total: 32 bytes (50% wasted!)

```

## 📊 Before Optimization

### Code

```go
type BadUser struct {
    ID     int32    // 4 bytes @ offset 0
    Active bool     // 1 byte  @ offset 4
    Name   string   // 16 bytes @ offset 8 (3 bytes padding)
    Age    int8     // 1 byte  @ offset 24 (7 bytes padding)
} // 32 bytes total

```

### Performance Metrics

| **Metric** | **Value** | **Notes** |
| --- | --- | --- |
| Struct Size | 32 bytes | 12 bytes wasted (37.5%) |
| Memory for 1M users | 32.00 MB |  |
| Allocation Time (1M) | ~120ms | May vary by system |
| Cache Efficiency | Poor | Padding reduces locality |

### Cost Impact (Before)

- **Memory:** 32.00 MB per 1M users
- **AWS t3.medium (8GB):** Can hold ~250M BadUsers
- **Monthly Cost:** $0.0293 per 1M users (memory only)

## ⚡ Optimization

### Solution

Reorder struct fields from largest to smallest to minimize padding.

**Key Changes:**

1. **Follow the type size hierarchy:** Slice → String → 64-bit → 32-bit → 16-bit → 8-bit
2. **8-byte alignment:** Keep 8-byte types at 8-byte offsets
3. **Pack small types:** Place bools, int8, int16 together

### Optimized Code

```go
// ✅ GOOD: Fields ordered from largest to smallest
type GoodUser struct {
    // 8-byte aligned types first
    Name   string   // 16 bytes @ offset 0 (highest priority)

    // 4-byte types next
    ID     int32    // 4 bytes @ offset 16

    // 1-byte types last (packed together)
    Age    int8     // 1 byte  @ offset 20
    Active bool     // 1 byte  @ offset 21

    // 2 bytes padding @ offset 22-23
} // 24 bytes total (0 wasted!)

```

**Alternative ordering (if you want to keep ID first):**

```go
// ✅ ALSO GOOD: Still follows large-to-small principle
type GoodUserAlt struct {
    Name   string   // 16 bytes @ offset 0
    ID     int64    // 8 bytes @ offset 16 (changed to int64 for better alignment)
    Age    int8     // 1 byte  @ offset 24
    Active bool     // 1 byte  @ offset 25
    // 6 bytes padding @ offset 26-31
} // 32 bytes, but better than original if int64 is needed

```

## 📈 After Optimization

### Performance Metrics

| **Metric** | **Value** | **Improvement** |
| --- | --- | --- |
| Struct Size | 24 bytes | 25% reduction |
| Memory for 1M users | 24.00 MB | 8.00 MB saved |
| Allocation Time (1M) | ~85ms | 30% faster |
| Cache Efficiency | Better | Improved locality |

### Cost Impact (After)

- **Memory:** 24.00 MB per 1M users
- **AWS t3.medium (8GB):** Can hold ~341M GoodUsers (36% more!)
- **Monthly Cost:** $0.0220 per 1M users

## 💰 Total Cost Savings

**For 1 Million Users:**

```
Memory Before: 32.00 MB
Memory After:  24.00 MB
Memory Saved:  8.00 MB

Cost per GB-month: $3.75
Monthly Savings:   $0.0293 → $0.0220 = $0.0073 (25%)
Annual Savings:   $0.0876 per 1M users

```

**Scaling Projections:**

- **10M users:** $0.88/year savings
- **100M users:** $8.76/year savings
- **1B users:** $87.60/year savings

**Note:** These are memory-only savings. Additional benefits include:

- Reduced GC pressure → lower CPU costs
- Better cache performance → faster response times
- Lower memory bandwidth → better scalability

## 🧪 How to Run

### Prerequisites

```bash
# Install Go 1.21+
go version

# Navigate to day-01
cd day-01

```

### Run the Demo

```bash
go run main.go

```

### Run Benchmarks

```bash
# Quick benchmark
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Compare with benchstat (install: go install golang.org/x/perf/cmd/benchstat@latest)
go test -bench=. -count=5 | benchstat -

```

### Run Tests

```bash
go test -v

```

## 📊 Visualization

### Memory Layout Diagram:

```
BAD USER (32 bytes):
┌────┬────────┬──────────┬──────┬─────┬───────┐
│ ID │ Active │  Padding │ Name │ Age │Padding│
│ 4B │   1B   │    3B    │  16B │ 1B  │  7B   │
└────┴────────┴──────────┴──────┴─────┴───────┘

GOOD USER (24 bytes):
┌──────────────┬─────┬─────┬──────┬─────────┐
│     Name     │ ID  │ Age │Active│ Padding │
│     16B      │ 4B  │ 1B  │  1B  │   2B    │
└──────────────┴─────┴─────┴──────┴─────────┘

```

### Type Size Hierarchy Visual Guide:

```
OPTIMAL FIELD ORDERING:

      [SLICES]           [STRINGS]           [64-bit]          [32-bit]         [Small]
        ↓                   ↓                   ↓                 ↓               ↓
      []T (24B)          string (16B)       int64 (8B)       int32 (4B)       bool (1B)
      map[K]V*          interface{}       float64 (8B)      float32 (4B)      int8 (1B)
      chan T*             any (16B)         *T (8B)          rune (4B)        byte (1B)
                                           func() (8B)
                                                                             int16 (2B)
* = 8B pointer in struct, actual data in heap

```

## 📚 Learnings

### Key Insights

1. **Go adds padding automatically** based on field sizes and order
2. **Follow the hierarchy:** Slice → String → 64-bit → 32-bit → 16-bit → 8-bit
3. **8-byte types need 8-byte alignment** - they should start at offsets divisible by 8
4. **Group small fields together** to fill padding gaps

### Quick Reference: Struct Field Ordering

```go
// ✅ PERFECT ORDERING TEMPLATE:
type OptimizedStruct struct {
    // 1. Slices first (24 bytes)
    Items []Item

    // 2. Strings and interfaces (16 bytes)
    Name string
    Data interface{}

    // 3. 64-bit values (8 bytes)
    ID      int64
    Balance float64
    Next    *OptimizedStruct  // pointer

    // 4. 32-bit values (4 bytes)
    Age    int32
    Score  float32

    // 5. 16-bit values (2 bytes)
    Code   int16
    Status uint16

    // 6. 8-bit and bool values LAST (1 byte)
    Active bool
    Flag   byte
    Value  int8
}

```

### When to Apply This Optimization

✅ **DO apply when:**

- Struct is instantiated millions of times
- Memory usage is a bottleneck
- Working with in-memory databases/caches
- Building high-performance APIs

❌ **DON'T over-optimize when:**

- Struct is rarely instantiated
- Readability would suffer significantly
- Working with protobuf/gRPC (field order matters for compatibility)
- The struct has < 10 instances

### Practical Tips

1. **Use `unsafe.Sizeof()`** to measure struct sizes
2. **Check field offsets** with `unsafe.Offsetof()`
3. **Use `fieldalignment` tool** to find optimization opportunities:
    
    ```bash
    # Install and run fieldalignment
    go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
    fieldalignment ./...
    
    ```
    
4. **Profile memory usage** in production with `pprof`
5. **Remember the rule:** "Slice first, bool last"

## 🔗 References & Further Reading

### Documentation

- [Go Memory Layout](https://go101.org/article/memory-layout.html)
- [The Go Memory Model](https://go.dev/ref/mem)
- [unsafe package](https://pkg.go.dev/unsafe)

### Articles

- [Padding is Hard](https://qvault.io/golang/golang-memory-allocation/)
- [Go Struct Memory Optimization](https://medium.com/@felipedutratine/go-struct-memory-optimization-48e9c044ea64)

### Tools

- [structlayout](https://github.com/dominikh/go-tools/tree/master/cmd/structlayout) - Visualize struct layouts
- [fieldalignment](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/fieldalignment) - Find structs that could be packed better

## 🚀 Next Steps

### Immediate Actions

1. **Run this code** and see the results on your machine
2. **Find similar structs** in your codebase using `fieldalignment`
3. **Apply optimization** to at least one production struct
4. **Measure the impact** with benchmarks

### Follow-up Exploration

1. **Slice vs Array performance** — see [slice-performance](../slice-performance/)
2. **Investigate** how this affects JSON marshaling/unmarshaling
3. **Explore** memory pooling techniques
4. **Learn about** escape analysis and heap vs stack allocation

---

**🎯 Challenge Complete!** You've saved 25% memory with proper field ordering.

**Memory Rule to Remember:** "Slice (24B) → String (16B) → 64-bit (8B) → 32-bit (4B) → bool (1B) last"
