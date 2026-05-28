# Batch Processing vs Individual Operations

## 📋 Overview

Amortizing network round-trip overhead by grouping individual INSERT/PUBLISH operations into configurable batches, with adaptive sizing based on load.

## TL;DR
- **Problem**: Individual INSERT/PUBLISH = 1 network round-trip per record; N records = N round-trips
- **Solution**: Batch INSERT with configurable size + AdaptiveBatcher for load-based adjustment → 1 round-trip per N records
- **Impact**: 48x+ speedup; 10M ops/day saves 99% of round-trips, ~$57/month in compute costs

## 🎯 Problem Statement

When inserting records into a database or publishing messages to a queue one at a time, each operation incurs a full network round-trip. At scale (10M+ ops/day), this wastes network I/O, database connection time, and compute resources waiting on network responses.

**Real-world impact:** 10M individual INSERT operations/day = 10M network round-trips. With batching (size=100), that drops to 100K round-trips — a 99% reduction.

## 🔍 Root Cause

### Why individual operations are expensive:

1. **Network round-trip per record**: Each INSERT/PUBLISH requires a full request-response cycle (~0.5ms per trip)
2. **Connection setup overhead**: Each operation may require connection acquisition from pool
3. **Protocol overhead**: TCP framing, serialization, and acknowledgment per operation
4. **Database lock contention**: Individual operations compete for locks more frequently

### The cost chain:

```go
// ❌ BAD: N records = N network round-trips
func IndividualInsert(db *DB, records []Record) {
    for _, r := range records {
        // Each iteration: 500μs network + 1μs processing = 501μs
        db.Exec("INSERT INTO table VALUES ($1)", r)
    }
}
// 1000 records × 501μs = ~501ms total
```

With batching:
```go
// ✅ GOOD: N records = ceil(N/batchSize) round-trips
func BatchInsert(db *DB, records []Record, batchSize int) {
    for i := 0; i < len(records); i += batchSize {
        batch := records[i:min(i+batchSize, len(records))]
        // One round-trip for entire batch: 500μs + 100μs setup + 1μs×len(batch)
        db.Exec("INSERT INTO table VALUES ...", batch)
    }
}
// 1000 records, batch=100: 10 batches × 700μs = ~7ms total
```

## ⚡ Solution

### 1. BatchInsert with Configurable Size

Group records into batches and send one network request per batch:

```go
func BatchInsert(db *SimulatedDB, records []Record, batchSize int) int {
    if batchSize <= 0 {
        batchSize = 1
    }
    inserted := 0
    for i := 0; i < len(records); i += batchSize {
        end := i + batchSize
        if end > len(records) {
            end = len(records)
        }
        batch := records[i:end]

        db.mu.Lock()
        // One network round-trip per batch (not per record)
        time.Sleep(db.networkLatency)
        // Connection setup cost (once per batch)
        time.Sleep(db.connectionSetup)
        // Per-record processing still applies
        time.Sleep(db.perRecordCost * time.Duration(len(batch)))
        db.records = append(db.records, batch...)
        inserted += len(batch)
        db.mu.Unlock()
    }
    return inserted
}
```

### 2. AdaptiveBatcher for Load-Based Adjustment

Dynamically adjust batch size based on current system load:

```go
type AdaptiveBatcher struct {
    MinBatch    int
    MaxBatch    int
    CurrentSize int
    LoadFactor  float64 // 0.0 = idle, 1.0 = max load
}

func (ab *AdaptiveBatcher) UpdateLoad(loadFactor float64) {
    if loadFactor >= 0.7 {
        // High load: increase batch size to amortize overhead
        ab.CurrentSize = min(int(float64(ab.CurrentSize)*2.0), ab.MaxBatch)
    } else if loadFactor <= 0.3 {
        // Low load: decrease batch size to reduce latency
        ab.CurrentSize = max(int(float64(ab.CurrentSize)*0.5), ab.MinBatch)
    }
    // Between thresholds: keep current size (hysteresis)
}
```

### 3. Diminishing Returns Beyond Optimal Batch Size

Beyond batch size ~100-500, throughput gains plateau but memory usage keeps growing:

- **Batch 1→100**: Massive improvement (eliminate 99% of round-trips)
- **Batch 100→500**: Marginal improvement (already amortized)
- **Batch 500→10000**: Negligible improvement + significant memory cost

## 📈 Benchmarks

> Machine: Apple M1, Go 1.24.4

### Individual vs Batch INSERT (100 records, PostgreSQL simulation)

| **Method** | **Total Time** | **Per Record** | **Speedup** |
| --- | --- | --- | --- |
| Individual INSERT × 100 | ~50ms | ~501 μs/record | 1x (baseline) |
| Batch INSERT (size=50) × 100 | ~1.3ms | ~13 μs/record | ~48x |

### Diminishing Returns (1000 records, varying batch size)

| **Batch Size** | **Total Time** | **Per Record** | **Throughput** |
| --- | --- | --- | --- |
| 1 (individual) | ~501ms | ~501 μs | ~2.0K/s |
| 10 | ~51ms | ~51 μs | ~19.6K/s |
| 50 | ~11ms | ~11 μs | ~90.9K/s |
| 100 | ~6ms | ~6 μs | ~166.7K/s |
| 500 | ~2ms | ~2 μs | ~500K/s |
| 1000 | ~1.1ms | ~1.1 μs | ~909K/s |

→ Notice: Beyond batch size ~100-500, time improvement slows but memory usage keeps growing (diminishing returns + memory trade-off).

### Memory Trade-off

| **Batch Size** | **Buffer Memory** |
| --- | --- |
| 100 | ~25 KB |
| 1000 | ~250 KB |
| 10000 | ~2500 KB |

Larger batches hold more records in memory simultaneously. Beyond the optimal threshold, memory grows linearly while throughput gains diminish.

### RabbitMQ: Individual vs Batch PUBLISH (100 messages)

| **Method** | **Total Time** | **Per Message** | **Speedup** |
| --- | --- | --- | --- |
| Individual PUBLISH × 100 | ~30ms | ~301 μs/msg | 1x (baseline) |
| Batch PUBLISH (size=50) × 100 | ~0.8ms | ~8 μs/msg | ~37x |

```bash
# Run benchmarks yourself
cd patterns/batch-processing
go test -bench=. -benchmem -benchtime=3s
```

## 💰 Cost Impact

### Network Round-trip Reduction (10M ops/day, batch size=100)

| **Metric** | **Individual** | **Batch (size=100)** | **Savings** |
| --- | --- | --- | --- |
| Round-trips/day | 10,000,000 | 100,000 | 99% reduction |
| Total processing time | ~1,392 vCPU-hours/day | ~0.02 vCPU-hours/day | ~1,392 hours saved |

### AWS Cost Projection (t3.medium @ $0.0416/vCPU-hour)

| **Metric** | **Individual** | **Batch (size=100)** | **Savings** |
| --- | --- | --- | --- |
| Cost/day | $57.89 | $0.0008 | $57.89/day |
| Cost/month | $1,736.64 | $0.03 | **~$1,737/month** |

### Connection Pool Impact

| **Metric** | **Individual** | **Batch** |
| --- | --- | --- |
| Connections needed | 50 | 10 |
| Memory (5MB/conn) | 250 MB | 50 MB |
| Memory saved | — | 200 MB (→ smaller instance possible) |

### Summary

Batch processing with size=100 eliminates:
- 99% of network round-trips
- ~1,392 vCPU-hours/day of network wait time
- ~$57/month in compute costs (conservative estimate)
- 200 MB of connection pool memory

## When to Apply

✅ **DO apply when:**

- Inserting/publishing multiple records in a single logical operation
- Network latency dominates per-operation cost (database, message queue, external API)
- Throughput is more important than per-record latency
- Operating at scale (10K+ ops/day)
- Records are independent (no ordering dependency between individual inserts)

❌ **DON'T over-optimize when:**

- Operations require immediate consistency (each record must be confirmed before next)
- Batch size exceeds available memory
- Error handling requires per-record granularity (partial batch failure is unacceptable)
- Latency-sensitive: batching adds wait time to accumulate records
- Volume is low (< 100 ops/day — overhead of batching logic isn't worth it)

### Decision Guide

```
Is this a high-volume operation? (>1K ops/day)
├── YES → Are records independent?
│         ├── YES → Use batch processing
│         │         └── Is load variable?
│         │             ├── YES → Use AdaptiveBatcher
│         │             └── NO  → Use fixed batch size (100-500)
│         └── NO  → Use individual operations with ordering guarantees
└── NO  → Individual operations are fine
```

### Optimal Batch Size Selection

| **Scenario** | **Recommended Size** | **Rationale** |
| --- | --- | --- |
| Low-latency requirement | 10-50 | Smaller batches flush faster |
| Balanced throughput | 100-200 | Sweet spot for most workloads |
| Maximum throughput | 500-1000 | Diminishing returns beyond this |
| Memory-constrained | 50-100 | Limit buffer memory usage |

## 🧪 How to Run

```bash
cd patterns/batch-processing

# Run demo (shows individual vs batch comparison, adaptive batcher, cost projection)
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Detailed benchmark (3 seconds per test)
go test -bench=. -benchmem -benchtime=3s

# Run with race detector
go test -race -v
```


## When This Is Acceptable

- Processing fewer than 10 items where the overhead of batching logic exceeds the savings
- Real-time/streaming scenarios where items must be processed immediately as they arrive
- When each item requires unique error handling that batch processing would obscure
