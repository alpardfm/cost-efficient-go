# Interface Dispatch vs Concrete Type

## TL;DR
- **Problem**: Interface method calls prevent compiler inlining, adding ~1-3ns/call overhead
- **Solution**: "Interface at boundary, concrete internally" — accept interfaces for flexibility, use concrete types in hot loops
- **Impact**: Negligible for APIs (5 calls/request = noise), measurable only in tight loops (1M+ iterations)

## Problem

Go interfaces provide powerful abstraction for clean architecture, dependency injection, and testability. However, calling methods through an interface adds indirect dispatch overhead and prevents the Go compiler from inlining the method body.

For most production code (REST APIs, CRUD services), this overhead is completely irrelevant. But in tight computational loops running millions of iterations, the accumulated overhead becomes measurable.

## Root Cause

When you call a method on a concrete type, the Go compiler knows exactly which function to call at compile time. This enables:
1. **Inlining** — the method body is inserted directly at the call site
2. **Constant folding** — compile-time evaluation of known values
3. **Dead code elimination** — removing unreachable branches

When you call through an interface, the compiler cannot determine the concrete type at compile time. It must:
1. **Look up the itab** (interface table) to find the method pointer
2. **Make an indirect call** through the function pointer
3. **Skip inlining** — the method body is unknown at compile time

```go
// ✅ Compiler CAN inline — knows exact type
func HotLoopConcrete(p *ConcreteProcessor, iterations int) float64 {
    var result float64
    for i := range iterations {
        result += p.Process(float64(i)) // inlined!
    }
    return result
}

// ❌ Compiler CANNOT inline — type unknown at compile time
func HotLoopInterface(p ProcessorInterface, iterations int) float64 {
    var result float64
    for i := range iterations {
        result += p.Process(float64(i)) // indirect call via itab
    }
    return result
}
```

Verify with: `go build -gcflags="-m" ./patterns/interface-dispatch/`

## Solution

Use the **"interface at boundary, concrete internally"** pattern:

- **API boundary**: Accept interfaces for flexibility, DI, and testability
- **Internal hot path**: Use concrete types for compiler optimization

```go
type Service struct {
    // Internal: concrete type for hot path performance
    processor *ConcreteProcessor
}

// Boundary: accept interface (flexible, testable)
func NewService(p ProcessorInterface) *Service {
    if concrete, ok := p.(*ConcreteProcessor); ok {
        return &Service{processor: concrete}
    }
    return &Service{processor: NewConcreteProcessor(1.0, 0.0)}
}

// Internal hot path: uses concrete type directly — no interface overhead
func (s *Service) ProcessBatch(values []float64) float64 {
    return s.processor.BatchProcess(values)
}
```

## Benchmarks

> Machine: Apple M1, Go 1.24.4

### Single Method Call

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `Process_Concrete` | 2.26 | 0 | 0 |
| `Process_Interface` | 2.29 | 0 | 0 |

Overhead per single call: **~0.03ns** — effectively unmeasurable.

### Hot Loop (1M iterations)

| Benchmark | ns/op | Overhead |
|-----------|------:|------:|
| `HotLoop_Concrete` (1M) | 6,355,442 | — |
| `HotLoop_Interface` (1M) | 8,236,506 | +30% |

Per-call overhead in tight loop: **~1.9ns/call** (inlining prevented).

### Scaling with Iteration Count

| Iterations | Concrete (ns/op) | Interface (ns/op) | Overhead |
|-----------:|------------------:|------------------:|---------:|
| 10K | 47,117 | 48,961 | +4% |
| 100K | 414,408 | 591,574 | +43% |
| 1M | 6,355,442 | 8,236,506 | +30% |
| 10M | 38,092,967 | 94,772,850 | +149% |

### Boundary Pattern (interface at boundary, concrete internally)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `BatchProcess_Concrete` (1M values) | 3,302,723 | 0 | 0 |
| `BatchProcess_Interface` (1M values) | 3,360,573 | 0 | 0 |
| `BoundaryPattern` (1M values) | 3,361,035 | 0 | 0 |

The boundary pattern achieves **near-concrete performance** because the interface dispatch happens only once (at the boundary), while the internal loop runs on the concrete type.

## Cost Impact

**Typical interface overhead: ~1-3ns per call**

| Scenario | Calls/Day | Interface Cost/Day | Concrete Cost/Day | Savings/Day |
|----------|----------:|-------------------:|------------------:|------------:|
| REST API (5 calls/req, 10M req) | 50M | $0.000001 | $0.000000 | $0.000001 |
| Stream processor (100 calls/op, 10M ops) | 1B | $0.000023 | $0.000005 | $0.000018 |
| Tight loop (1M calls/op, 10M ops) | 10T | $0.023148 | $0.004622 | $0.018526 |

> Cost basis: $0.0416/vCPU-hour (t3.medium)

**Verdict:**
- REST APIs: **$0.0004/year** — literally unmeasurable, use interfaces freely
- Stream processors: **$0.007/year** — still negligible
- Tight computation loops: **$6.76/year** — consider concrete types in inner loops

## When to Apply

✅ **Use concrete types when:**
- Loop runs 1M+ iterations on a hot path
- Profiling shows the method call is a bottleneck
- The function is CPU-bound computation (math, data processing)
- You've verified with `gcflags="-m"` that inlining is blocked

❌ **Use interfaces freely when:**
- Loop runs < 10K iterations (overhead is noise)
- Code is I/O-bound (network, disk, database)
- Clean architecture and testability matter more
- You're building API handlers, middleware, or service layers

### Decision Rule

```
If loop iterations < 10,000:
    → Use interfaces. Overhead is unmeasurable.

If loop iterations > 1,000,000 AND CPU-bound:
    → Use "interface at boundary, concrete internally"
    → Verify with profiling before optimizing
```

## How to Run

```bash
cd patterns/interface-dispatch

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Verify compiler inlining decisions
go build -gcflags="-m" .

# Verbose inlining analysis
go build -gcflags="-m -m" . 2>&1 | grep -E "(can inline|inlining|devirtualize)"
```


## When This Is Acceptable

- Almost always acceptable. The 1-3ns overhead per call is negligible for:
  - API handlers (network latency dominates)
  - Database operations (I/O dominates)
  - Any code path that isn't called millions of times per second in a tight loop
- Only consider concrete types in CPU-bound inner loops processing > 10M iterations
