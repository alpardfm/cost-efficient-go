# Cost-Efficient Go

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

A collection of Go performance optimization patterns — each with benchmarks, memory analysis, and real AWS cost projections.

Every pattern answers: **"How much money does this optimization save at scale?"**

---

## Patterns

| # | Pattern | Key Result | Link |
|---|---------|-----------|------|
| 1 | **Struct Alignment** | 25% memory reduction | [→](patterns/struct-alignment/) |
| 2 | **Slice Pre-allocation** | 4x faster, 91% fewer allocations | [→](patterns/slice-performance/) |
| 3 | **Map Internals & Overhead** | Understanding hidden memory costs | [→](patterns/map-internals/) |
| 4 | **JSON Processing Efficiency** | 2x faster batch, 77% less bandwidth | [→](patterns/json-processing/) |
| 5 | **Profiling & Benchmarking** | Correct measurement techniques, percentiles | [→](patterns/profiling-benchmarking/) |
| 6 | **Connection Pooling** | 2.7x faster, 40x less memory per request | [→](patterns/connection-pooling/) |
| 7 | **Query Optimization** | 4.4x faster SELECT, 50x faster with batch, O(1) pagination | [→](patterns/query-optimization/) |
| 8 | **HTTP Client Optimization** | 2.6x faster with body drain, timeout protection | [→](patterns/http-client-optimization/) |
| 9 | **Worker Pool Pattern** | Controlled concurrency, 99.9% less goroutine memory | [→](patterns/worker-pool/) |
| 10 | **Caching Strategies** | 21,872x faster cache hit, 99% DB load reduction | [→](patterns/caching-strategies/) |

---

## Each Pattern Includes

```
patterns/<name>/
├── main.go              # Implementation (before & after)
├── benchmark_test.go    # Go benchmarks with -benchmem
└── README.md            # Analysis: problem → solution → benchmark → cost impact
```

Every pattern follows the same structure:
1. **Problem** — what's inefficient and why
2. **Root Cause** — technical explanation
3. **Solution** — optimized implementation
4. **Benchmarks** — real numbers from `go test -bench`
5. **Cost Impact** — AWS cost projection at scale (per 1M/10M/100M units)

---

## Quick Start

```bash
git clone https://github.com/alpardfm/cost-efficient-go.git
cd cost-efficient-go

# Run a specific pattern
cd patterns/struct-alignment
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s
```

---

## Why This Exists

Most optimization guides tell you *what* to do. This project tells you **how much money it saves**.

Every pattern includes:
- Real benchmark numbers (not theoretical)
- Memory savings in bytes and percentages
- AWS cost projection at scale
- When to apply vs when to skip

This is engineering economics — making data-driven decisions about where optimization effort pays off.

---

## Cost Calculation Framework

Default assumptions for cost projections:
- AWS t3.medium: ~$30/month (8GB RAM)
- Cost per GB-month: $3.75
- Baseline: 1M → 10M → 100M → 1B units

Each pattern calculates:
```
Memory Before vs After → Savings per unit → Savings at scale → $/month saved
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
