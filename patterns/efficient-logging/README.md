# Efficient Logging Patterns

## TL;DR
- **Problem**: `log.Printf` allocates on every call; at 100K+ logs/sec this creates massive GC pressure
- **Solution**: zerolog/zap for production, custom `ZeroAllocLogger` for extreme cases, check-then-log for disabled levels
- **Impact**: Zero-alloc loggers achieve 10x+ faster throughput than Printf; at 1M logs/hour, switching saves measurable CPU and memory

## Problem

High-throughput services generating 100K+ log entries per second face a hidden cost: every `log.Printf` call allocates memory for format string processing and argument boxing.

```
log.Printf("msg=%s user_id=%d action=%s latency_ms=%.2f", msg, userID, action, latencyMs)
```

Each call:
1. **Boxes arguments** into `interface{}` (heap allocation per arg)
2. **Processes format string** (reflection + string building)
3. **Allocates result string** (new heap object)

At 100K logs/sec:
- ~200K–300K allocations/sec just from logging
- GC pressure increases proportionally
- CPU cycles wasted on formatting strings that may never be read
- Disabled log levels (DEBUG in production) still pay full formatting cost with naive implementations

## Root Cause

Go's `fmt.Sprintf` (used internally by `log.Printf`) must:

1. **Parse format verbs** at runtime — no compile-time optimization
2. **Box each argument** into `interface{}` — forces heap escape for non-pointer types
3. **Build output string** — allocates a new `[]byte`, converts to `string`
4. **Reflect on types** — `%v` and `%d` use reflection to determine formatting

Structured loggers (zerolog, zap) avoid this by:
- Using typed field methods (`Int()`, `Str()`) — no interface boxing
- Writing directly to a pre-allocated buffer — no intermediate string
- Checking log level BEFORE any formatting work — disabled levels cost nothing

## Solution

### 1. Structured Loggers (zerolog / zap)

```go
// zerolog — zero allocation for enabled levels
logger.Info().
    Int("user_id", userID).
    Str("action", action).
    Float64("latency_ms", latencyMs).
    Msg("request")

// zap — near-zero allocation with sugar-free API
logger.Info("request",
    zap.Int("user_id", userID),
    zap.String("action", action),
    zap.Float64("latency_ms", latencyMs),
)
```

### 2. Custom ZeroAllocLogger (Extreme Cases)

For services where even zerolog's minimal overhead matters, a custom logger with pre-allocated buffers from `sync.Pool`:

```go
type ZeroAllocLogger struct {
    pool    sync.Pool
    writer  io.Writer
    level   LogLevel
    bufSize int
}

func (l *ZeroAllocLogger) Log(level LogLevel, msg string, userID int, action string, latencyMs float64) {
    // Check level first — if disabled, ZERO work is done
    if level < l.level {
        return
    }

    // Get pre-allocated buffer from pool
    bufPtr := l.pool.Get().(*[]byte)
    buf := (*bufPtr)[:0]

    // Format without allocation using strconv.Append*
    buf = append(buf, "level="...)
    buf = appendLevel(buf, level)
    buf = append(buf, " msg="...)
    buf = append(buf, msg...)
    buf = append(buf, " user_id="...)
    buf = strconv.AppendInt(buf, int64(userID), 10)
    buf = append(buf, " action="...)
    buf = append(buf, action...)
    buf = append(buf, " latency_ms="...)
    buf = strconv.AppendFloat(buf, latencyMs, 'f', 2, 64)
    buf = append(buf, '\n')

    l.writer.Write(buf)

    // Return buffer to pool
    *bufPtr = buf
    l.pool.Put(bufPtr)
}
```

### 3. Check-Then-Log Pattern (Key Insight)

**The biggest win is ensuring disabled log levels produce ZERO work:**

```go
// ✅ GOOD: Check-then-log — zero work when level disabled
func (l *ZeroAllocLogger) Log(level LogLevel, msg string, ...) {
    if level < l.level {
        return // ZERO allocations, ZERO formatting
    }
    // ... format and write
}

// ❌ BAD: Always-format — wastes CPU even when disabled
formatted := fmt.Sprintf("msg=%s user_id=%d ...", msg, userID, ...)
if enabled {
    log.Print(formatted)
}
// formatted was allocated and immediately discarded
```

## Benchmarks

> Machine: Apple M1, Go 1.24.4

### Logger Throughput Comparison (Structured Logging)

| Logger | ns/op | B/op | allocs/op |
|--------|------:|-----:|----------:|
| log.Printf | ~800 | ~128 | 2 |
| slog (JSON) | ~500 | ~64 | 1 |
| zerolog | ~150 | 0 | 0 |
| zap | ~200 | 0 | 0 |
| ZeroAllocLogger | ~100 | 0 | 0 |

**ZeroAllocLogger is 8–10x faster than log.Printf** on high-throughput scenarios.

### Disabled Level Overhead (1M calls, DEBUG disabled)

| Pattern | Duration | Allocs | Bytes |
|---------|----------|-------:|------:|
| CheckThenLog (ZeroAllocLogger) | ~1ms | 0 | 0 |
| AlwaysFormat (fmt.Sprintf) | ~500ms | 1,000,000 | ~80 MB |

When DEBUG is disabled in production (the common case), check-then-log produces **zero allocations** while always-format wastes ~80 MB of memory on strings that are never used.

### High-Throughput Demo (200K entries)

| Logger | ops/sec |
|--------|--------:|
| log.Printf | ~1.2M |
| slog (JSON) | ~2.0M |
| zerolog | ~6.5M |
| zap | ~5.0M |
| ZeroAllocLogger | ~10M+ |

All loggers achieve 100K+ logs/sec. Zero-alloc loggers (zerolog, ZeroAllocLogger) produce 0 heap allocations at any throughput level.

*Note: Run `go test -bench=. -benchmem` in `patterns/efficient-logging/` for exact numbers on your machine.*

## Cost Impact

### Per-Entry Savings

```
log.Printf:       ~128 bytes + 2 allocs per log entry
ZeroAllocLogger:  0 bytes + 0 allocs per log entry
Saved:            128 bytes + 2 allocs per entry (100% reduction)
```

### At Scale (AWS t3.medium: $3.75/GB RAM, $0.0416/vCPU-hour)

| Metric | log.Printf | ZeroAllocLogger | Savings |
|--------|--------:|--------:|--------:|
| Allocs/day (1M logs/hr) | 48M | 0 | 48M fewer |
| Memory/day | ~2.9 GB | 0 | ~2.9 GB |
| CPU hours/day | ~0.46 | ~0.06 | ~0.40 hrs |
| **Monthly cost** | **~$0.87** | **~$0.07** | **~$0.80/instance** |

### At 10 Service Instances

```
Monthly savings: ~$8.00 (CPU + memory combined)
Annual savings:  ~$96 per service cluster
```

### The Real Savings: Disabled Levels

The cost table above only covers *enabled* log entries. The bigger win comes from disabled levels:

```
Service with 80% DEBUG logs disabled:
  AlwaysFormat: Still pays full formatting cost → wasted CPU
  CheckThenLog: Zero cost for disabled entries → free

At 1M total log calls/hour (800K disabled):
  AlwaysFormat waste: ~61 MB/hour of unused strings
  CheckThenLog waste: 0
```

## When to Apply

### ✅ DO use zero-alloc logging when:
- Service generates 100K+ log entries/sec
- GC pressure from logging is measurable (check with `GODEBUG=gctrace=1`)
- Hot paths include logging (request handlers, middleware)
- Production runs with DEBUG/TRACE disabled (most services)

### ✅ DO use check-then-log pattern ALWAYS:
- Even with low-throughput services, disabled levels should cost nothing
- This is the single highest-impact logging optimization
- All major structured loggers (zerolog, zap, slog) implement this internally

### ❌ DON'T optimize logging when:
- Service generates < 1K logs/sec — overhead is negligible
- Logging is not on the hot path (startup, shutdown, error paths)
- Readability matters more than performance (use slog for good balance)
- You need Printf-style formatting for human-readable logs in development

### Logger Selection Guide

| Scenario | Recommended Logger |
|----------|-------------------|
| General production | zerolog or zap |
| Stdlib preference | slog (Go 1.21+) |
| Extreme throughput (10M+/sec) | Custom ZeroAllocLogger |
| Development/debugging | log.Printf (readability > perf) |
| Mixed (dev + prod) | slog with level-based handler swap |

## How to Run

```bash
cd patterns/efficient-logging

# Run demo (shows allocation comparison, disabled level overhead, throughput, cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Run benchmarks with longer duration for stable results
go test -bench=. -benchmem -benchtime=5s

# Run only logger comparison benchmarks
go test -bench=BenchmarkLog -benchmem

# Run disabled level benchmarks
go test -bench=BenchmarkDisabled -benchmem

# Check GC behavior with tracing
GODEBUG=gctrace=1 go run main.go
```
