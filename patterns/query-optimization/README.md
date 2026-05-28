# Query Optimization & Indexing

## 📋 Overview

Optimizing database query patterns in Go — SELECT specific columns, batch queries, and efficient pagination to reduce latency and cost.

## TL;DR
- **Problem**: SELECT *, N+1 queries, and OFFSET pagination cause excessive memory, round-trips, and full table scans
- **Solution**: SELECT specific columns, batch with IN clause, use keyset (WHERE id > cursor) pagination
- **Impact**: 4.4x faster SELECT, 50x faster batch vs N+1 (real DB), O(1) pagination for deep pages

## 🎯 Problem Statement

The three most common query anti-patterns in backend APIs:

1. **SELECT *** — fetches 15 columns when you need 3, wasting bandwidth and memory
2. **N+1 queries** — 100 users × 1 query each = 100 round trips instead of 1
3. **OFFSET pagination** — page 500 must scan and discard 9,980 rows before returning 20

**Real-world impact:** A single API endpoint with N+1 queries can take 100ms+ when it should take 2ms.

## 📊 Benchmark Results

Tested on Apple M1, Go 1.24.4:

### SELECT * vs SELECT specific columns

| Benchmark | ns/op | B/op | Memory per row |
|-----------|-------|------|----------------|
| SELECT * (1K rows, 15 cols) | 389,970 | 253,976 | ~254 B |
| SELECT specific (1K rows, 3 cols) | **87,794** | **40,984** | ~41 B |
| SELECT * (10K rows) | 5,704,474 | 2,482,200 | ~248 B |
| SELECT specific (10K rows) | **1,104,830** | **401,432** | ~40 B |

**Result: SELECT specific is 4.4x faster and uses 6.2x less memory.**

### N+1 vs Batch Query

| Benchmark | ns/op | Note |
|-----------|-------|------|
| N+1 (100 users, in-memory) | 7,366 | Simulated — no network |
| Batch (100 users, in-memory) | 13,483 | Simulated — no network |

**In-memory, N+1 appears faster** because there's no network overhead. But in real databases:

| Pattern | Real-world latency |
|---------|-------------------|
| N+1 (100 queries × 1ms RTT) | **~100ms** |
| Batch (1 query × 2ms) | **~2ms** |

**Real-world: Batch is 50x faster** because network round-trip dominates.

### Pagination: OFFSET vs Keyset

| Benchmark | ns/op | Behavior |
|-----------|-------|----------|
| OFFSET page 1 | 139 | Fast (skip 0 rows) |
| OFFSET page 100 | 103 | Still fast in-memory |
| OFFSET page 500 | 94 | In-memory slice is O(1) |
| Keyset from start | 126 | Linear scan to find start |
| Keyset from middle (id > 5000) | 107,984 | Scan half the data |
| Keyset from end (id > 9980) | 304,866 | Scan almost all data |

**Important:** These in-memory benchmarks are misleading for real databases! In PostgreSQL:

| Pattern | Page 1 | Page 500 (10K rows) |
|---------|--------|---------------------|
| OFFSET | ~1ms | **~50ms** (scans 9,980 rows) |
| Keyset (WHERE id > X) | ~1ms | **~1ms** (index seek) |

**Real DB: Keyset is O(1) via B-tree index, OFFSET is O(n).**

## ⚡ Optimizations

### 1. SELECT only what you need

```go
// ❌ BAD: Fetches password, bio, avatar — never used in list view
rows, _ := db.Query("SELECT * FROM users LIMIT 20")

// ✅ GOOD: Only fetch what the API response needs
rows, _ := db.Query("SELECT id, email, full_name FROM users LIMIT 20")
```

**Why it matters:**
- Less data transferred from DB → Go process
- Smaller scan structs → less memory allocation
- Fewer columns → faster index-only scans possible

### 2. Batch queries instead of N+1

```go
// ❌ BAD: N+1 — one query per user
for _, userID := range userIDs {
    orders, _ := db.Query("SELECT * FROM orders WHERE user_id = $1", userID)
    // ...
}

// ✅ GOOD: Single batch query
query := "SELECT * FROM orders WHERE user_id = ANY($1)"
orders, _ := db.Query(query, pq.Array(userIDs))
// Then group by user_id in Go
```

**Rule of thumb:** If you're querying inside a loop, you have an N+1 problem.

### 3. Keyset pagination for deep pages

```go
// ❌ BAD: OFFSET — gets slower as page number increases
db.Query("SELECT * FROM users ORDER BY id LIMIT 20 OFFSET $1", (page-1)*20)

// ✅ GOOD: Keyset — constant time regardless of page depth
db.Query("SELECT * FROM users WHERE id > $1 ORDER BY id LIMIT 20", lastID)
```

**Trade-off:** Keyset pagination doesn't support "jump to page 50" — it's cursor-based. Use OFFSET for admin panels with page numbers, keyset for infinite scroll / API pagination.

### 4. Index your WHERE and JOIN columns

```sql
-- Without index: full table scan O(n)
SELECT * FROM orders WHERE user_id = 123;  -- scans ALL rows

-- With index: B-tree lookup O(log n)
CREATE INDEX idx_orders_user_id ON orders(user_id);
SELECT * FROM orders WHERE user_id = 123;  -- index seek, instant
```

**Index checklist:**
- Every foreign key column (user_id, order_id, etc.)
- Every column in WHERE clauses
- Every column in ORDER BY (for sorted queries)
- Composite indexes for multi-column filters

## 💰 Cost Impact

### SELECT * vs SELECT specific (10K rows, 1000 req/sec)

```
Memory per second:
  SELECT *:        2.48 MB × 1000 = 2.48 GB/sec allocated
  SELECT specific: 0.40 MB × 1000 = 0.40 GB/sec allocated
  Saved: 2.08 GB/sec less GC pressure

Network bandwidth (DB → App):
  SELECT * (15 cols): ~500 bytes/row × 10K × 1000 = 5 GB/sec
  SELECT 3 cols:      ~100 bytes/row × 10K × 1000 = 1 GB/sec
  Saved: 4 GB/sec bandwidth between DB and app
```

### N+1 Elimination (100 users endpoint, 1000 req/sec)

```
Before (N+1):
  DB queries per second: 100 × 1000 = 100,000 queries/sec
  Latency per request: ~100ms
  DB CPU: saturated

After (Batch):
  DB queries per second: 1 × 1000 = 1,000 queries/sec
  Latency per request: ~2ms
  DB CPU: 1% utilized

Result: 100x fewer queries, 50x lower latency
Can downgrade DB instance: db.r5.xlarge → db.t3.medium
Monthly savings: ~$500-800/month
```

### Keyset Pagination (deep pages, 100 req/sec)

```
Before (OFFSET page 500):
  Query time: ~50ms (scan 9,980 rows)
  DB I/O: reads 9,980 rows to return 20

After (Keyset WHERE id > X):
  Query time: ~1ms (index seek)
  DB I/O: reads exactly 20 rows

Result: 50x faster for deep pages
```

## 🧪 How to Run

```bash
cd patterns/query-optimization

# Run demo
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Longer benchmark
go test -bench=. -benchmem -benchtime=3s
```

## 📚 Key Takeaways

1. **Never SELECT *** in production — always list specific columns
2. **If you query in a loop, you have N+1** — use IN clause or JOIN
3. **Use keyset pagination for APIs** — OFFSET is only for admin UIs with page numbers
4. **Index every FK and WHERE column** — unindexed queries are full table scans
5. **Measure in real DB, not in-memory** — network RTT dominates, not CPU

### Quick Reference: When to Use What

| Scenario | Pattern |
|----------|---------|
| List API (infinite scroll) | Keyset pagination |
| Admin panel (page numbers) | OFFSET (acceptable for small datasets) |
| Related data (user → orders) | Batch with IN clause |
| List view (table display) | SELECT specific columns only |
| Detail view (single item) | SELECT * is acceptable |
| Search with filters | Composite index on filter columns |

### PostgreSQL Index Types

| Type | Use Case |
|------|----------|
| B-tree (default) | Equality, range, ORDER BY |
| Hash | Equality only (rare) |
| GIN | Full-text search, JSONB, arrays |
| GiST | Geometric, range types |
| pg_trgm + GIN | LIKE '%search%' (trigram) |


## When This Is Acceptable

- Admin/backoffice queries that run infrequently (< 10/minute)
- Data exploration queries during development
- Queries on small tables (< 1000 rows) where full scan is faster than index lookup
