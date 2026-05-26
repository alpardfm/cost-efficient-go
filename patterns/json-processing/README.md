# JSON Processing Efficiency

## 📋 Overview

Optimizing JSON serialization in Go by leveraging `omitempty`, typed structs over maps, and understanding the hidden costs of `encoding/json` reflection.

## 🎯 Problem Statement

Go's `encoding/json` uses reflection for every marshal/unmarshal call. Poor struct design amplifies this cost:

- Empty fields still get serialized (wasted bytes over the wire)
- `map[string]interface{}` forces runtime type inspection per key
- Untyped data causes extra allocations during unmarshal

**Real-world impact:** A REST API returning 1000 products with `map[string]interface{}` attributes uses **2x more CPU** and **2.7x more memory** than typed structs.

## 🔍 Root Cause Analysis

### Why `map[string]interface{}` is expensive:

1. **Runtime type assertion** — each value needs type checking at marshal time
2. **Map iteration is unordered** — Go must sort keys for deterministic output
3. **Interface boxing** — every value stored in `interface{}` requires heap allocation
4. **No struct tag caching** — maps can't benefit from `encoding/json`'s tag cache

### Why missing `omitempty` wastes bandwidth:

```json
// Without omitempty — 189 bytes
{"status":"success","message":"","data":{"id":"123"},"error":"","meta":{"page":0,"page_size":0,"total":0,"total_pages":0,"request_id":"","version":""}}

// With omitempty — 42 bytes  
{"status":"success","data":{"id":"123"}}
```

That's **77% less data** per response for the common case (success without pagination).

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4:

### Response Marshaling

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| BadResponse (no omitempty) | 2,108 | 480 | 7 |
| GoodResponse (with meta) | 2,886 | 400 | 7 |
| GoodResponse (minimal, omitempty) | **1,658** | **208** | **5** |

**Key insight:** When most fields are empty (common case for create/update responses), `omitempty` gives **21% faster** marshaling and **57% less memory**.

### Map vs Struct (Product attributes)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Marshal map product | 2,286 | 384 | **9** |
| Marshal struct product | **1,224** | **160** | **2** |
| Unmarshal map product | 4,945 | 776 | **21** |
| Unmarshal struct product | **4,557** | **392** | **10** |

**Struct is 1.9x faster to marshal with 77% fewer allocations.**

### Batch Processing (1000 products)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| 1000 map products | 1,421,078 | 227,084 | **5,002** |
| 1000 struct products | **690,532** | **83,011** | **2** |

**At batch scale: 2x faster, 63% less memory, 2500x fewer allocations.**

## ⚡ Optimizations Applied

### 1. Use `omitempty` for optional fields

```go
// ❌ BAD: Always serializes empty fields
type Response struct {
    Status  string      `json:"status"`
    Message string      `json:"message"`
    Error   string      `json:"error"`
    Meta    Meta        `json:"meta"`
}

// ✅ GOOD: Skips zero-value fields
type Response struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    Error   string `json:"error,omitempty"`
    Meta    *Meta  `json:"meta,omitempty"`  // pointer + omitempty = skip when nil
}
```

### 2. Use typed structs instead of `map[string]interface{}`

```go
// ❌ BAD: Runtime type inspection per key
type Product struct {
    ID         int                    `json:"id"`
    Attributes map[string]interface{} `json:"attributes"`
}

// ✅ GOOD: Compile-time type safety, cached struct tags
type Product struct {
    ID    int           `json:"id"`
    Attrs *ProductAttrs `json:"attributes,omitempty"`
}

type ProductAttrs struct {
    Color  string `json:"color,omitempty"`
    Size   string `json:"size,omitempty"`
    Weight int    `json:"weight,omitempty"`
}
```

### 3. Use pointer for optional nested structs

```go
// ❌ BAD: Meta always serialized even when empty
Meta Meta `json:"meta"`

// ✅ GOOD: Meta skipped entirely when nil
Meta *Meta `json:"meta,omitempty"`
```

## 💰 Cost Impact

### Per-Request Savings (minimal response)

```
Bytes over wire:  189 → 42 bytes (77% reduction)
Marshal time:     2,108 → 1,658 ns (21% faster)
Memory per call:  480 → 208 bytes (57% less)
```

### At Scale: 100K requests/day

```
Bandwidth Before: 189 bytes × 100,000 = 18.9 MB/day
Bandwidth After:  42 bytes × 100,000 = 4.2 MB/day
Bandwidth Saved:  14.7 MB/day = 441 MB/month

Data transfer cost (AWS): $0.09/GB
Monthly savings: 0.441 × $0.09 = $0.04/month (bandwidth only)
```

### At Scale: Batch API (1000 products, 10K calls/day)

```
CPU time Before: 1.42ms × 10,000 = 14.2 seconds/day
CPU time After:  0.69ms × 10,000 = 6.9 seconds/day
CPU saved: 7.3 seconds/day of pure JSON processing

Memory Before: 227 KB × 10,000 = 2.27 GB/day allocated
Memory After:  83 KB × 10,000 = 0.83 GB/day allocated
Memory saved: 1.44 GB/day less GC pressure
```

**Note:** The real cost savings compound with GC pressure reduction. Less allocations = less GC pauses = more consistent latency = better P99.

## 🧪 How to Run

```bash
cd patterns/json-processing

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark
go test -bench=. -benchmem -benchtime=3s
```

## 📚 Key Takeaways

1. **Always use `omitempty`** for fields that are frequently empty/zero
2. **Use pointer + omitempty** for optional nested structs (skips entire subtree)
3. **Prefer typed structs over `map[string]interface{}`** — 2x faster, 77% fewer allocs
4. **The savings compound at batch scale** — 1000 items shows 2500x fewer allocations
5. **Bandwidth savings matter** for high-traffic APIs — 77% less data per response

### When to Apply

✅ **DO apply when:**
- API responses have many optional fields
- You're serializing lists/batches of objects
- Response payload is a significant portion of latency
- You have known attribute schemas (not truly dynamic)

❌ **DON'T apply when:**
- Data is genuinely dynamic/unknown at compile time
- You're prototyping and schema is unstable
- The endpoint handles <100 requests/day (optimization won't matter)
