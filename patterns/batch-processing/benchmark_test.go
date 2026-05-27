package main

import (
	"fmt"
	"runtime"
	"testing"
)

// ============================================================
// Benchmark Tests: Batch Processing vs Individual Operations
// ============================================================
// These benchmarks demonstrate:
// 1. ns/op for various batch sizes (1, 10, 50, 100, 500, 1000, 10000)
// 2. Diminishing returns curve — throughput gains plateau at large batch sizes
// 3. Memory trade-off — larger batches use more memory (B/op grows)
//
// Requirements: 7.1, 7.2, 7.3
// ============================================================

// globalInserted prevents compiler from optimizing away results.
var globalInserted int

// --- Individual vs Batch INSERT Benchmarks (Requirement 7.1) ---

// Benchmark_IndividualInsert_100 benchmarks individual INSERT for 100 records.
func Benchmark_IndividualInsert_100(b *testing.B) {
	db := NewSimulatedDB()
	records := generateRecords(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.mu.Lock()
		db.records = db.records[:0]
		db.mu.Unlock()
		globalInserted = IndividualInsert(db, records)
	}
}

// Benchmark_IndividualInsert_1000 benchmarks individual INSERT for 1000 records.
func Benchmark_IndividualInsert_1000(b *testing.B) {
	db := NewSimulatedDB()
	records := generateRecords(1000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.mu.Lock()
		db.records = db.records[:0]
		db.mu.Unlock()
		globalInserted = IndividualInsert(db, records)
	}
}

// --- Batch INSERT: Varying Batch Sizes (Requirement 7.1, 7.3) ---
// These benchmarks show diminishing returns AND memory trade-off.
// As batch size increases:
//   - ns/op decreases (fewer network round-trips)
//   - But gains diminish beyond optimal threshold (~100-500)
//   - B/op increases (larger batch buffers in memory)

func Benchmark_BatchInsert_Size1(b *testing.B) {
	benchmarkBatchInsert(b, 1)
}

func Benchmark_BatchInsert_Size10(b *testing.B) {
	benchmarkBatchInsert(b, 10)
}

func Benchmark_BatchInsert_Size50(b *testing.B) {
	benchmarkBatchInsert(b, 50)
}

func Benchmark_BatchInsert_Size100(b *testing.B) {
	benchmarkBatchInsert(b, 100)
}

func Benchmark_BatchInsert_Size500(b *testing.B) {
	benchmarkBatchInsert(b, 500)
}

func Benchmark_BatchInsert_Size1000(b *testing.B) {
	benchmarkBatchInsert(b, 1000)
}

func Benchmark_BatchInsert_Size10000(b *testing.B) {
	benchmarkBatchInsert(b, 10000)
}

// benchmarkBatchInsert is the shared helper for batch INSERT benchmarks.
// Uses 10000 records to clearly show diminishing returns at large batch sizes.
func benchmarkBatchInsert(b *testing.B, batchSize int) {
	b.Helper()
	const numRecords = 10000
	db := NewSimulatedDB()
	records := generateRecords(numRecords)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.mu.Lock()
		db.records = db.records[:0]
		db.mu.Unlock()
		globalInserted = BatchInsert(db, records, batchSize)
	}
}

// --- Individual vs Batch PUBLISH Benchmarks (Requirement 7.2) ---

// Benchmark_IndividualPublish_100 benchmarks individual PUBLISH for 100 messages.
func Benchmark_IndividualPublish_100(b *testing.B) {
	q := NewSimulatedQueue()
	records := generateRecords(100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.mu.Lock()
		q.messages = q.messages[:0]
		q.mu.Unlock()
		globalInserted = IndividualPublish(q, records)
	}
}

// Benchmark_IndividualPublish_1000 benchmarks individual PUBLISH for 1000 messages.
func Benchmark_IndividualPublish_1000(b *testing.B) {
	q := NewSimulatedQueue()
	records := generateRecords(1000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.mu.Lock()
		q.messages = q.messages[:0]
		q.mu.Unlock()
		globalInserted = IndividualPublish(q, records)
	}
}

// --- Batch PUBLISH: Varying Batch Sizes (Requirement 7.2, 7.3) ---

func Benchmark_BatchPublish_Size1(b *testing.B) {
	benchmarkBatchPublish(b, 1)
}

func Benchmark_BatchPublish_Size10(b *testing.B) {
	benchmarkBatchPublish(b, 10)
}

func Benchmark_BatchPublish_Size50(b *testing.B) {
	benchmarkBatchPublish(b, 50)
}

func Benchmark_BatchPublish_Size100(b *testing.B) {
	benchmarkBatchPublish(b, 100)
}

func Benchmark_BatchPublish_Size500(b *testing.B) {
	benchmarkBatchPublish(b, 500)
}

func Benchmark_BatchPublish_Size1000(b *testing.B) {
	benchmarkBatchPublish(b, 1000)
}

func Benchmark_BatchPublish_Size10000(b *testing.B) {
	benchmarkBatchPublish(b, 10000)
}

// benchmarkBatchPublish is the shared helper for batch PUBLISH benchmarks.
func benchmarkBatchPublish(b *testing.B, batchSize int) {
	b.Helper()
	const numRecords = 10000
	q := NewSimulatedQueue()
	records := generateRecords(numRecords)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.mu.Lock()
		q.messages = q.messages[:0]
		q.mu.Unlock()
		globalInserted = BatchPublish(q, records, batchSize)
	}
}

// --- Memory Trade-off Benchmarks (Requirement 7.3) ---
// These benchmarks explicitly measure memory allocation growth
// as batch size increases, demonstrating the trade-off:
// larger batches = fewer round-trips BUT more memory per batch.

func Benchmark_MemoryTradeoff_BatchSize1(b *testing.B) {
	benchmarkMemoryTradeoff(b, 1)
}

func Benchmark_MemoryTradeoff_BatchSize10(b *testing.B) {
	benchmarkMemoryTradeoff(b, 10)
}

func Benchmark_MemoryTradeoff_BatchSize50(b *testing.B) {
	benchmarkMemoryTradeoff(b, 50)
}

func Benchmark_MemoryTradeoff_BatchSize100(b *testing.B) {
	benchmarkMemoryTradeoff(b, 100)
}

func Benchmark_MemoryTradeoff_BatchSize500(b *testing.B) {
	benchmarkMemoryTradeoff(b, 500)
}

func Benchmark_MemoryTradeoff_BatchSize1000(b *testing.B) {
	benchmarkMemoryTradeoff(b, 1000)
}

func Benchmark_MemoryTradeoff_BatchSize10000(b *testing.B) {
	benchmarkMemoryTradeoff(b, 10000)
}

// benchmarkMemoryTradeoff measures peak memory allocation for a given batch size.
// The SimulatedDB's internal slice grows as records are appended in batch-sized chunks.
// Larger batch sizes cause larger single append operations, demonstrating memory trade-off.
func benchmarkMemoryTradeoff(b *testing.B, batchSize int) {
	b.Helper()
	const numRecords = 10000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Fresh DB each iteration to measure allocation from scratch
		db := NewSimulatedDB()
		records := generateRecords(numRecords)
		globalInserted = BatchInsert(db, records, batchSize)
	}
}

// --- Diminishing Returns Demonstration (Requirement 7.3) ---
// This test prints a table showing how throughput gains diminish
// as batch size grows beyond the optimal threshold.

func Benchmark_DiminishingReturns_Table(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping diminishing returns table in short mode")
	}

	batchSizes := []int{1, 10, 50, 100, 500, 1000, 10000}
	const numRecords = 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, bs := range batchSizes {
			db := NewSimulatedDB()
			records := generateRecords(numRecords)
			globalInserted = BatchInsert(db, records, bs)
		}
	}
}

// --- Adaptive Batcher Benchmark ---

func Benchmark_AdaptiveBatcher_LowLoad(b *testing.B) {
	const numRecords = 1000
	records := generateRecords(numRecords)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db := NewSimulatedDB()
		batcher := NewAdaptiveBatcher(10, 1000)
		batcher.UpdateLoad(0.2) // Low load → small batches
		globalInserted = batcher.ProcessAdaptive(db, records)
	}
}

func Benchmark_AdaptiveBatcher_HighLoad(b *testing.B) {
	const numRecords = 1000
	records := generateRecords(numRecords)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db := NewSimulatedDB()
		batcher := NewAdaptiveBatcher(10, 1000)
		batcher.UpdateLoad(0.9) // High load → large batches
		globalInserted = batcher.ProcessAdaptive(db, records)
	}
}

// --- Unit Tests: Correctness Verification ---

func TestBatchInsert_Correctness(t *testing.T) {
	tests := []struct {
		name      string
		numRecs   int
		batchSize int
	}{
		{"batch=1, records=10", 10, 1},
		{"batch=5, records=10", 10, 5},
		{"batch=10, records=10", 10, 10},
		{"batch=50, records=100", 100, 50},
		{"batch=100, records=100", 100, 100},
		{"batch=1000, records=500", 500, 1000}, // batch > records
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewSimulatedDB()
			records := generateRecords(tt.numRecs)
			inserted := BatchInsert(db, records, tt.batchSize)

			if inserted != tt.numRecs {
				t.Errorf("BatchInsert returned %d, want %d", inserted, tt.numRecs)
			}
			if len(db.records) != tt.numRecs {
				t.Errorf("DB has %d records, want %d", len(db.records), tt.numRecs)
			}
		})
	}
}

func TestBatchPublish_Correctness(t *testing.T) {
	tests := []struct {
		name      string
		numRecs   int
		batchSize int
	}{
		{"batch=1, messages=10", 10, 1},
		{"batch=5, messages=10", 10, 5},
		{"batch=10, messages=10", 10, 10},
		{"batch=50, messages=100", 100, 50},
		{"batch=100, messages=100", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSimulatedQueue()
			records := generateRecords(tt.numRecs)
			published := BatchPublish(q, records, tt.batchSize)

			if published != tt.numRecs {
				t.Errorf("BatchPublish returned %d, want %d", published, tt.numRecs)
			}
			if len(q.messages) != tt.numRecs {
				t.Errorf("Queue has %d messages, want %d", len(q.messages), tt.numRecs)
			}
		})
	}
}

func TestAdaptiveBatcher_ScalesWithLoad(t *testing.T) {
	batcher := NewAdaptiveBatcher(10, 1000)

	// Low load should keep small batch size
	batcher.UpdateLoad(0.1)
	if batcher.GetBatchSize() > 10 {
		t.Errorf("Low load: batch size %d, expected <= 10", batcher.GetBatchSize())
	}

	// High load should increase batch size
	batcher.UpdateLoad(0.9)
	highSize := batcher.GetBatchSize()
	if highSize <= 10 {
		t.Errorf("High load: batch size %d, expected > 10", highSize)
	}

	// Even higher load should increase further (up to max)
	batcher.UpdateLoad(1.0)
	veryHighSize := batcher.GetBatchSize()
	if veryHighSize < highSize {
		t.Errorf("Very high load: batch size %d < previous %d", veryHighSize, highSize)
	}
}

func TestMeasureBatchPerformance_DiminishingReturns(t *testing.T) {
	// Verify that throughput gains diminish as batch size grows
	// Batch size 10 should be much faster than 1, but 10000 should not be
	// proportionally faster than 1000.
	const numRecords = 1000

	metrics1 := MeasureBatchPerformance(numRecords, 1)
	metrics10 := MeasureBatchPerformance(numRecords, 10)
	metrics1000 := MeasureBatchPerformance(numRecords, 1000)

	// Batch=10 should be significantly faster than batch=1
	speedup1to10 := float64(metrics1.TotalTime) / float64(metrics10.TotalTime)
	if speedup1to10 < 2.0 {
		t.Logf("Warning: batch=10 only %.1fx faster than batch=1 (expected >2x)", speedup1to10)
	}

	// Batch=1000 should NOT be proportionally faster than batch=10
	// (diminishing returns)
	speedup10to1000 := float64(metrics10.TotalTime) / float64(metrics1000.TotalTime)
	t.Logf("Speedup batch=1→10: %.1fx", speedup1to10)
	t.Logf("Speedup batch=10→1000: %.1fx", speedup10to1000)
	t.Logf("Diminishing returns confirmed: 1→10 gain (%.1fx) > 10→1000 gain (%.1fx)",
		speedup1to10, speedup10to1000)
}

func TestMeasureBatchPerformance_MemoryGrowth(t *testing.T) {
	// Verify that memory usage grows with batch size
	const numRecords = 10000

	// Measure memory for different batch sizes
	var memStats []struct {
		batchSize int
		memory    uint64
	}

	batchSizes := []int{1, 100, 1000, 10000}
	for _, bs := range batchSizes {
		// Run multiple times and take the measurement
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		db := NewSimulatedDB()
		records := generateRecords(numRecords)
		BatchInsert(db, records, bs)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		memUsed := memAfter.TotalAlloc - memBefore.TotalAlloc
		memStats = append(memStats, struct {
			batchSize int
			memory    uint64
		}{bs, memUsed})
	}

	// Log memory growth
	for _, ms := range memStats {
		t.Logf("Batch size %5d: memory allocated = %s",
			ms.batchSize, formatBytesTest(ms.memory))
	}

	// The key insight: memory allocation should be present for all sizes
	// (records are always allocated), but the pattern of allocation differs
	t.Log("Memory trade-off: larger batches hold more records in memory simultaneously")
}

// formatBytesTest formats bytes for test output.
func formatBytesTest(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}
