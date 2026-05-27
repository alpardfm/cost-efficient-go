package query_optimization

import (
	"testing"
	"time"
)

// ============================================================
// Benchmarks: Query Pattern Comparisons
// ============================================================

var (
	benchUsers   = generateUsers(10000)
	benchUserIDs = func() []int {
		ids := make([]int, 100)
		for i := range ids {
			ids[i] = i + 1
		}
		return ids
	}()
	benchOrders = generateOrders(benchUserIDs, 5)
	benchSink   interface{}
)

// --- SELECT * vs SELECT specific ---

func BenchmarkSelectStar1K(b *testing.B) {
	data := benchUsers[:1000]
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateSelectStar(data)
	}
}

func BenchmarkSelectSpecific1K(b *testing.B) {
	data := benchUsers[:1000]
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateSelectSpecific(data)
	}
}

func BenchmarkSelectStar10K(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateSelectStar(benchUsers)
	}
}

func BenchmarkSelectSpecific10K(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateSelectSpecific(benchUsers)
	}
}

// --- N+1 vs Batch ---

func BenchmarkNPlusOne100Users(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateNPlusOne(benchUserIDs, benchOrders)
	}
}

func BenchmarkBatchQuery100Users(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateBatchQuery(benchUserIDs, benchOrders)
	}
}

// --- N+1 vs Batch with Simulated Network Latency ---
// These benchmarks include time.Sleep to simulate real network round-trip latency.
// This makes the N+1 problem clearly visible: 100 queries × 1ms = ~100ms vs 1 query × 2ms = ~2ms.

func BenchmarkNPlusOneWithLatency100Users(b *testing.B) {
	latency := 1 * time.Millisecond
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateNPlusOneWithLatency(benchUserIDs, benchOrders, latency)
	}
}

func BenchmarkBatchQueryWithLatency100Users(b *testing.B) {
	// Batch query is slightly heavier (2ms) but only 1 round-trip
	latency := 2 * time.Millisecond
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateBatchQueryWithLatency(benchUserIDs, benchOrders, latency)
	}
}

// --- Pagination ---

func BenchmarkOffsetPage1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateOffsetPagination(benchUsers, 1, 20)
	}
}

func BenchmarkOffsetPage100(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateOffsetPagination(benchUsers, 100, 20)
	}
}

func BenchmarkOffsetPage500(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateOffsetPagination(benchUsers, 500, 20)
	}
}

func BenchmarkKeysetFromStart(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateKeysetPagination(benchUsers, 0, 20)
	}
}

func BenchmarkKeysetFromMiddle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateKeysetPagination(benchUsers, 5000, 20)
	}
}

func BenchmarkKeysetFromEnd(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SimulateKeysetPagination(benchUsers, 9980, 20)
	}
}

// --- IN Clause Building ---

func BenchmarkBuildINClause10(b *testing.B) {
	ids := benchUserIDs[:10]
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink, _ = BuildINClause(ids)
	}
}

func BenchmarkBuildINClause100(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink, _ = BuildINClause(benchUserIDs)
	}
}
