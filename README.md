# Cost-Efficient Go

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)
![Version](https://img.shields.io/badge/Version-v2.0.0-blue?style=flat-square)

An importable Go library for detecting cost-efficiency anti-patterns in Go source code via AST-based static analysis. Each of the 20 detectors identifies a specific performance issue and returns structured findings with severity, category, explanation, and suggested fixes.

---

## Installation

```bash
go get github.com/alpardfm/cost-efficient-go@v2.0.0
```

---

## Quick Start

```go
package main

import (
    "fmt"
    "go/ast"

    "github.com/alpardfm/cost-efficient-go/registry"
    "github.com/alpardfm/cost-efficient-go/types"
)

func main() {
    // Get all registered detectors
    detectors := registry.AllDetectors()
    fmt.Printf("Loaded %d detectors\n", len(detectors))

    // Look up a specific detector
    d, ok := registry.DetectorByID("CEG-001")
    if ok {
        fmt.Printf("Found: %s - %s\n", d.Rule().ID, d.Rule().Name)
    }

    // Filter by category or severity
    memoryDetectors := registry.DetectorsByCategory(types.Memory)
    criticalDetectors := registry.DetectorsBySeverity(types.Critical)
    fmt.Printf("Memory: %d, Critical: %d\n", len(memoryDetectors), len(criticalDetectors))

    // Run detection on an AST node
    ctx := types.ASTContext{
        FilePath:    "example.go",
        Line:        42,
        Node:        &ast.Ident{Name: "example"},
        CodeContext: `s = append(s, item)`,
    }

    for _, det := range detectors {
        findings := det.Detect(ctx)
        for _, f := range findings {
            fmt.Printf("[%s] %s:%d - %s\n", f.Severity, f.FilePath, f.Line, f.Explanation)
        }
    }
}
```

---

## Architecture

```
github.com/alpardfm/cost-efficient-go/
├── types/                          # Core types: Rule, Finding, ASTContext, Detector interface
├── registry/                       # Aggregates all 20 detectors with lookup functions
└── patterns/
    ├── batch-processing/           # CEG-001: Batch vs sequential I/O
    ├── caching-strategies/         # CEG-002: Cache miss detection
    ├── channel-patterns/           # CEG-003: Unbuffered channel usage
    ├── connection-pooling/         # CEG-004: Connection-per-request anti-pattern
    ├── context-cancellation/       # CEG-005: Missing context cancellation
    ├── efficient-logging/          # CEG-006: Allocating logger calls
    ├── error-handling/             # CEG-007: Inefficient error patterns
    ├── goroutine-leak/             # CEG-008: Leaked goroutines
    ├── http-client-optimization/   # CEG-009: HTTP client misuse
    ├── interface-dispatch/         # CEG-010: Interface overhead awareness
    ├── json-processing/            # CEG-011: Inefficient JSON handling
    ├── map-internals/              # CEG-012: Map memory overhead
    ├── profiling-benchmarking/     # CEG-013: Measurement anti-patterns
    ├── query-optimization/         # CEG-014: N+1 queries, missing indexes
    ├── redis-pipeline/             # CEG-015: Non-pipelined Redis calls
    ├── slice-performance/          # CEG-016: Slice growth without pre-allocation
    ├── string-building/            # CEG-017: String concatenation with +
    ├── struct-alignment/           # CEG-018: Struct padding waste
    ├── sync-pool/                  # CEG-019: Missing object pooling
    └── worker-pool/                # CEG-020: Unbounded goroutine spawning
```

---

## API Reference

### types package

```go
import "github.com/alpardfm/cost-efficient-go/types"
```

| Type | Description |
|------|-------------|
| `Detector` | Interface with `Detect(ctx ASTContext) []Finding` and `Rule() Rule` |
| `ASTContext` | Input struct: FilePath, Line, Node (ast.Node), CodeContext |
| `Finding` | Output struct: RuleID, FilePath, Line, Explanation, SuggestedFix, Severity, Category, CodeContext |
| `Rule` | Metadata: ID, Name, Description, Severity, Category, Suggestion, ReferenceLinks |
| `Severity` | Constants: `Minor`, `Major`, `Critical` |
| `Category` | Constants: `Memory`, `Concurrency`, `IO`, `ErrorHandling` |

### registry package

```go
import "github.com/alpardfm/cost-efficient-go/registry"
```

| Function | Description |
|----------|-------------|
| `AllDetectors() []types.Detector` | Returns all 20 registered detectors |
| `DetectorByID(id string) (types.Detector, bool)` | Lookup by rule ID (e.g. "CEG-001") |
| `DetectorsByCategory(cat types.Category) []types.Detector` | Filter by category |
| `DetectorsBySeverity(sev types.Severity) []types.Detector` | Filter by severity |

All registry functions are safe for concurrent use.

---

## Detectors

| ID | Pattern | Category | Severity |
|----|---------|----------|----------|
| CEG-001 | Batch Processing | IO | Major |
| CEG-002 | Caching Strategies | Memory | Major |
| CEG-003 | Channel Patterns | Concurrency | Minor |
| CEG-004 | Connection Pooling | IO | Critical |
| CEG-005 | Context Cancellation | Concurrency | Critical |
| CEG-006 | Efficient Logging | Memory | Minor |
| CEG-007 | Error Handling | ErrorHandling | Major |
| CEG-008 | Goroutine Leak | Concurrency | Critical |
| CEG-009 | HTTP Client Optimization | IO | Major |
| CEG-010 | Interface Dispatch | Memory | Minor |
| CEG-011 | JSON Processing | Memory | Major |
| CEG-012 | Map Internals | Memory | Minor |
| CEG-013 | Profiling & Benchmarking | Memory | Minor |
| CEG-014 | Query Optimization | IO | Critical |
| CEG-015 | Redis Pipeline | IO | Major |
| CEG-016 | Slice Performance | Memory | Major |
| CEG-017 | String Building | Memory | Major |
| CEG-018 | Struct Alignment | Memory | Minor |
| CEG-019 | Sync Pool | Memory | Major |
| CEG-020 | Worker Pool | Concurrency | Major |

---

## Design Principles

- **Stateless detectors** — each `Detect` call is a pure function of its input, inherently safe for concurrent use
- **No file I/O** — detectors receive pre-parsed AST context; the consuming analyzer handles parsing
- **No panics** — detectors handle nil nodes and unexpected AST structures gracefully
- **Immutable registry** — all detectors register at init time; the registry is read-only after startup

---

## Using Individual Patterns

You can import specific pattern packages directly:

```go
import (
    "github.com/alpardfm/cost-efficient-go/patterns/batch-processing"
    "github.com/alpardfm/cost-efficient-go/types"
)

func main() {
    detector := batch_processing.NewDetector()
    findings := detector.Detect(ctx)
}
```

---

## Examples

Each pattern includes educational example code in its `examples/` subdirectory:

```bash
# Run a specific pattern's example
go run ./patterns/struct-alignment/examples/

# Run benchmarks for a pattern
go test -bench=. -benchmem ./patterns/slice-performance/
```

---

## Running Tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Property-based tests only
go test -run TestProperty ./registry/
```

---

## Versioning

- **v1.0.0** — Educational repository (20 patterns as standalone `package main` programs)
- **v2.0.0** — Importable library with Detector interface, registry, and structured types

ID allocation:
- `CEG-001` to `CEG-099`: Reserved for core patterns
- `CEG-1000+`: Designated for custom/external detectors

---

## License

MIT License. See [LICENSE](LICENSE) for details.
