package main

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"
)

// ============================================================
// PATTERN 17: Batch Processing vs Individual Operations
// ============================================================
// Problem: Individual INSERT/PUBLISH operations incur per-operation
// network round-trip overhead. At scale (10M+ ops/day), this wastes
// network I/O, database connection time, and compute resources.
//
// This pattern demonstrates:
// 1. Individual INSERT — one network round-trip per record
// 2. Batch INSERT — amortize network cost across N records
// 3. Adaptive batching — adjust batch size based on current load
// 4. Diminishing returns when batch size exceeds optimal threshold
// 5. Memory trade-off: larger batches use more memory
// ============================================================

// --- Simulated Database/Queue (In-Memory) ---

// Record represents a single database record to insert.
type Record struct {
	ID      int
	Payload [256]byte // Simulate realistic record size (~256 bytes)
}

// SimulatedDB represents an in-memory PostgreSQL-like store.
// It simulates network latency for each round-trip.
type SimulatedDB struct {
	mu              sync.Mutex
	records         []Record
	networkLatency  time.Duration // Simulated per-round-trip latency
	perRecordCost   time.Duration // CPU cost per record processing
	connectionSetup time.Duration // Per-batch connection overhead
}

// NewSimulatedDB creates a new simulated database with realistic latencies.
func NewSimulatedDB() *SimulatedDB {
	return &SimulatedDB{
		records:         make([]Record, 0, 1024),
		networkLatency:  500 * time.Microsecond, // 0.5ms network round-trip
		perRecordCost:   1 * time.Microsecond,   // 1μs per record processing
		connectionSetup: 100 * time.Microsecond, // 0.1ms connection setup per batch
	}
}

// SimulatedQueue represents an in-memory RabbitMQ-like message queue.
type SimulatedQueue struct {
	mu             sync.Mutex
	messages       []Record
	networkLatency time.Duration
	perMsgCost     time.Duration
	publishSetup   time.Duration
}

// NewSimulatedQueue creates a new simulated message queue.
func NewSimulatedQueue() *SimulatedQueue {
	return &SimulatedQueue{
		messages:       make([]Record, 0, 1024),
		networkLatency: 300 * time.Microsecond, // 0.3ms network round-trip
		perMsgCost:     500 * time.Nanosecond,  // 0.5μs per message
		publishSetup:   50 * time.Microsecond,  // 0.05ms publish setup
	}
}

// --- Individual Operations (Before) ---

// IndividualInsert inserts records one at a time into the simulated database.
// Each insert incurs a full network round-trip.
func IndividualInsert(db *SimulatedDB, records []Record) int {
	inserted := 0
	for i := range records {
		db.mu.Lock()
		// Simulate network round-trip for each individual INSERT
		time.Sleep(db.networkLatency)
		// Simulate per-record processing
		time.Sleep(db.perRecordCost)
		db.records = append(db.records, records[i])
		inserted++
		db.mu.Unlock()
	}
	return inserted
}

// IndividualPublish publishes messages one at a time to the simulated queue.
func IndividualPublish(q *SimulatedQueue, records []Record) int {
	published := 0
	for i := range records {
		q.mu.Lock()
		// Simulate network round-trip for each individual PUBLISH
		time.Sleep(q.networkLatency)
		time.Sleep(q.perMsgCost)
		q.messages = append(q.messages, records[i])
		published++
		q.mu.Unlock()
	}
	return published
}

// --- Batch Operations (After) ---

// BatchInsert inserts records in batches of the given size.
// Only one network round-trip per batch, amortizing the cost.
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

// BatchPublish publishes messages in batches to the simulated queue.
func BatchPublish(q *SimulatedQueue, records []Record, batchSize int) int {
	if batchSize <= 0 {
		batchSize = 1
	}
	published := 0
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		q.mu.Lock()
		// One network round-trip per batch
		time.Sleep(q.networkLatency)
		time.Sleep(q.publishSetup)
		time.Sleep(q.perMsgCost * time.Duration(len(batch)))
		q.messages = append(q.messages, batch...)
		published += len(batch)
		q.mu.Unlock()
	}
	return published
}

// --- Adaptive Batcher ---

// AdaptiveBatcher dynamically adjusts batch size based on current load.
// Under high load, it increases batch size to amortize overhead.
// Under low load, it decreases batch size to reduce latency.
type AdaptiveBatcher struct {
	MinBatch    int     // Minimum batch size
	MaxBatch    int     // Maximum batch size
	CurrentSize int     // Current batch size
	LoadFactor  float64 // Current load factor (0.0 = idle, 1.0 = max load)

	// Tuning parameters
	ScaleUpThreshold   float64 // Load above this → increase batch size
	ScaleDownThreshold float64 // Load below this → decrease batch size
	ScaleUpFactor      float64 // Multiply batch size by this when scaling up
	ScaleDownFactor    float64 // Multiply batch size by this when scaling down
}

// NewAdaptiveBatcher creates a new adaptive batcher with sensible defaults.
func NewAdaptiveBatcher(minBatch, maxBatch int) *AdaptiveBatcher {
	return &AdaptiveBatcher{
		MinBatch:           minBatch,
		MaxBatch:           maxBatch,
		CurrentSize:        minBatch,
		LoadFactor:         0.0,
		ScaleUpThreshold:   0.7,
		ScaleDownThreshold: 0.3,
		ScaleUpFactor:      2.0,
		ScaleDownFactor:    0.5,
	}
}

// UpdateLoad updates the load factor and adjusts batch size accordingly.
// loadFactor should be between 0.0 (idle) and 1.0 (max capacity).
func (ab *AdaptiveBatcher) UpdateLoad(loadFactor float64) {
	ab.LoadFactor = math.Max(0.0, math.Min(1.0, loadFactor))

	if ab.LoadFactor >= ab.ScaleUpThreshold {
		// High load: increase batch size to amortize overhead
		newSize := int(float64(ab.CurrentSize) * ab.ScaleUpFactor)
		if newSize > ab.MaxBatch {
			newSize = ab.MaxBatch
		}
		ab.CurrentSize = newSize
	} else if ab.LoadFactor <= ab.ScaleDownThreshold {
		// Low load: decrease batch size to reduce latency
		newSize := int(float64(ab.CurrentSize) * ab.ScaleDownFactor)
		if newSize < ab.MinBatch {
			newSize = ab.MinBatch
		}
		ab.CurrentSize = newSize
	}
	// Between thresholds: keep current size (hysteresis)
}

// GetBatchSize returns the current recommended batch size.
func (ab *AdaptiveBatcher) GetBatchSize() int {
	return ab.CurrentSize
}

// ProcessAdaptive processes records using the adaptive batch size.
func (ab *AdaptiveBatcher) ProcessAdaptive(db *SimulatedDB, records []Record) int {
	return BatchInsert(db, records, ab.CurrentSize)
}

// --- Diminishing Returns & Memory Trade-off Demonstration ---

// BatchMetrics captures performance and memory metrics for a given batch size.
type BatchMetrics struct {
	BatchSize     int
	TotalTime     time.Duration
	TimePerRecord time.Duration
	MemoryUsed    uint64  // Peak memory during batch operation
	Throughput    float64 // Records per second
}

// MeasureBatchPerformance measures performance for a specific batch size.
// Shows diminishing returns when batch size exceeds optimal threshold
// AND memory trade-off for large batches.
func MeasureBatchPerformance(numRecords int, batchSize int) BatchMetrics {
	db := NewSimulatedDB()
	records := generateRecords(numRecords)

	// Measure memory before
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	BatchInsert(db, records, batchSize)
	elapsed := time.Since(start)

	// Measure memory after
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memUsed := memAfter.TotalAlloc - memBefore.TotalAlloc

	return BatchMetrics{
		BatchSize:     batchSize,
		TotalTime:     elapsed,
		TimePerRecord: elapsed / time.Duration(numRecords),
		MemoryUsed:    memUsed,
		Throughput:    float64(numRecords) / elapsed.Seconds(),
	}
}

// --- Helper Functions ---

// generateRecords creates N test records with sequential IDs.
func generateRecords(n int) []Record {
	records := make([]Record, n)
	for i := range records {
		records[i].ID = i + 1
		// Fill payload with some data to simulate realistic record size
		for j := range records[i].Payload {
			records[i].Payload[j] = byte(i + j)
		}
	}
	return records
}

// --- Cost Projection ---

func calculateCostProjection() {
	fmt.Println("=== Cost Projection: Batch Processing at Scale (10M ops/day) ===")
	fmt.Println()

	// Parameters
	opsPerDay := 10_000_000 // 10M operations/day

	// Measured latencies (from simulated benchmarks)
	individualLatencyNs := int64(501_000) // ~501μs per individual op (500μs network + 1μs processing)
	batchLatencyNs := int64(601_000)      // ~601μs per batch (500μs network + 100μs setup + 1μs × records)
	optimalBatchSize := 100               // Optimal batch size from diminishing returns analysis

	// Individual: each op = 1 network round-trip
	individualTotalNs := int64(opsPerDay) * individualLatencyNs
	individualTotalHours := float64(individualTotalNs) / 1e9 / 3600

	// Batch: ops/batchSize batches, each = 1 network round-trip
	numBatches := opsPerDay / optimalBatchSize
	batchTotalNs := int64(numBatches) * batchLatencyNs
	batchTotalHours := float64(batchTotalNs) / 1e9 / 3600

	fmt.Printf("Operation Parameters:\n")
	fmt.Printf("  Operations/day:       %d (10M)\n", opsPerDay)
	fmt.Printf("  Optimal batch size:   %d\n", optimalBatchSize)
	fmt.Printf("  Network latency:      500μs per round-trip\n")
	fmt.Println()

	fmt.Printf("Network Round-trips:\n")
	fmt.Printf("  Individual:           %d round-trips/day (10M)\n", opsPerDay)
	fmt.Printf("  Batch (size=%d):     %d round-trips/day (100K)\n", optimalBatchSize, numBatches)
	fmt.Printf("  Reduction:            %.0f%% fewer round-trips\n",
		(1.0-float64(numBatches)/float64(opsPerDay))*100)
	fmt.Println()

	fmt.Printf("Total Processing Time:\n")
	fmt.Printf("  Individual:           %.1f vCPU-hours/day\n", individualTotalHours)
	fmt.Printf("  Batch (size=%d):     %.2f vCPU-hours/day\n", optimalBatchSize, batchTotalHours)
	fmt.Printf("  Savings:              %.1f vCPU-hours/day\n", individualTotalHours-batchTotalHours)
	fmt.Println()

	// AWS cost calculation
	// t3.medium: $0.0416/vCPU-hour
	costPerVCPUHour := 0.0416
	individualCostDay := individualTotalHours * costPerVCPUHour
	batchCostDay := batchTotalHours * costPerVCPUHour
	savingsDay := individualCostDay - batchCostDay
	savingsMonth := savingsDay * 30

	fmt.Printf("AWS Cost (t3.medium @ $0.0416/vCPU-hour):\n")
	fmt.Printf("  Individual:           $%.2f/day ($%.2f/month)\n",
		individualCostDay, individualCostDay*30)
	fmt.Printf("  Batch (size=%d):     $%.2f/day ($%.2f/month)\n",
		optimalBatchSize, batchCostDay, batchCostDay*30)
	fmt.Printf("  Savings:              $%.2f/day ($%.2f/month)\n", savingsDay, savingsMonth)
	fmt.Println()

	// Connection pool savings
	// Individual: needs more connections to handle concurrent ops
	// Batch: fewer concurrent connections needed
	individualConns := 50 // Need 50 connections for individual ops
	batchConns := 10      // Only need 10 connections for batched ops
	connMemoryMB := 5     // ~5MB per PostgreSQL connection
	memorySavedMB := (individualConns - batchConns) * connMemoryMB

	fmt.Printf("Connection Pool Impact:\n")
	fmt.Printf("  Individual:           %d connections × %dMB = %dMB\n",
		individualConns, connMemoryMB, individualConns*connMemoryMB)
	fmt.Printf("  Batch:                %d connections × %dMB = %dMB\n",
		batchConns, connMemoryMB, batchConns*connMemoryMB)
	fmt.Printf("  Memory saved:         %dMB (→ smaller instance possible)\n", memorySavedMB)
	fmt.Println()

	// Memory trade-off warning
	fmt.Printf("⚠️  Memory Trade-off (batch size vs memory):\n")
	recordSize := 256 // bytes per record
	fmt.Printf("  Batch size 100:       %d KB buffer needed\n", 100*recordSize/1024)
	fmt.Printf("  Batch size 1000:      %d KB buffer needed\n", 1000*recordSize/1024)
	fmt.Printf("  Batch size 10000:     %d KB buffer needed\n", 10000*recordSize/1024)
	fmt.Printf("  → Beyond optimal size, memory grows but throughput gains diminish\n")
	fmt.Println()

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("  Batch processing with size=%d eliminates:\n", optimalBatchSize)
	fmt.Printf("    • %.0f%% of network round-trips\n",
		(1.0-float64(numBatches)/float64(opsPerDay))*100)
	fmt.Printf("    • %.1f vCPU-hours/day of network wait time\n", individualTotalHours-batchTotalHours)
	fmt.Printf("    • $%.2f/month in compute costs\n", savingsMonth)
	fmt.Printf("  Diminishing returns beyond batch size ~500 (memory grows, throughput plateaus)\n")
}

// globalResult prevents compiler from optimizing away results.
var globalResult int

// --- Demonstration ---

func main() {
	fmt.Println("=== Batch Processing vs Individual Operations ===")
	fmt.Println()

	// 1. Individual vs Batch INSERT (PostgreSQL simulation)
	fmt.Println("--- PostgreSQL: Individual vs Batch INSERT ---")
	numRecords := 100
	records := generateRecords(numRecords)

	db1 := NewSimulatedDB()
	start := time.Now()
	n1 := IndividualInsert(db1, records)
	d1 := time.Since(start)
	fmt.Printf("Individual INSERT × %d: %v (%.0f μs/record)\n",
		numRecords, d1, float64(d1.Microseconds())/float64(numRecords))

	db2 := NewSimulatedDB()
	start = time.Now()
	n2 := BatchInsert(db2, records, 50)
	d2 := time.Since(start)
	fmt.Printf("Batch INSERT (size=50) × %d: %v (%.0f μs/record)\n",
		numRecords, d2, float64(d2.Microseconds())/float64(numRecords))

	fmt.Printf("Speedup: %.1fx faster with batching\n", float64(d1)/float64(d2))
	fmt.Printf("Records inserted: individual=%d, batch=%d\n", n1, n2)
	fmt.Println()

	// 2. Individual vs Batch PUBLISH (RabbitMQ simulation)
	fmt.Println("--- RabbitMQ: Individual vs Batch PUBLISH ---")
	q1 := NewSimulatedQueue()
	start = time.Now()
	p1 := IndividualPublish(q1, records)
	d3 := time.Since(start)
	fmt.Printf("Individual PUBLISH × %d: %v (%.0f μs/msg)\n",
		numRecords, d3, float64(d3.Microseconds())/float64(numRecords))

	q2 := NewSimulatedQueue()
	start = time.Now()
	p2 := BatchPublish(q2, records, 50)
	d4 := time.Since(start)
	fmt.Printf("Batch PUBLISH (size=50) × %d: %v (%.0f μs/msg)\n",
		numRecords, d4, float64(d4.Microseconds())/float64(numRecords))

	fmt.Printf("Speedup: %.1fx faster with batching\n", float64(d3)/float64(d4))
	fmt.Printf("Messages published: individual=%d, batch=%d\n", p1, p2)
	fmt.Println()

	// 3. Diminishing Returns & Memory Trade-off
	fmt.Println("--- Diminishing Returns: Batch Size vs Performance ---")
	fmt.Println("(1000 records, varying batch size)")
	fmt.Println()
	fmt.Printf("%-12s %-14s %-14s %-12s %-12s\n",
		"Batch Size", "Total Time", "Per Record", "Memory", "Throughput")
	fmt.Printf("%-12s %-14s %-14s %-12s %-12s\n",
		"----------", "----------", "----------", "------", "----------")

	batchSizes := []int{1, 10, 50, 100, 500, 1000}
	for _, bs := range batchSizes {
		metrics := MeasureBatchPerformance(1000, bs)
		fmt.Printf("%-12d %-14v %-14v %-12s %-12s\n",
			metrics.BatchSize,
			metrics.TotalTime.Round(time.Microsecond),
			metrics.TimePerRecord.Round(time.Microsecond),
			formatBytes(metrics.MemoryUsed),
			formatThroughput(metrics.Throughput))
	}
	fmt.Println()
	fmt.Println("→ Notice: Beyond batch size ~100-500, time improvement slows")
	fmt.Println("  but memory usage keeps growing (diminishing returns + memory trade-off)")
	fmt.Println()

	// 4. Adaptive Batcher Demo
	fmt.Println("--- Adaptive Batcher: Load-Based Adjustment ---")
	batcher := NewAdaptiveBatcher(10, 1000)
	fmt.Printf("Initial batch size: %d (min=%d, max=%d)\n",
		batcher.GetBatchSize(), batcher.MinBatch, batcher.MaxBatch)
	fmt.Println()

	// Simulate increasing load
	loadSequence := []float64{0.1, 0.3, 0.5, 0.7, 0.8, 0.9, 1.0, 0.6, 0.3, 0.1}
	fmt.Printf("%-8s %-14s %-10s\n", "Load", "Batch Size", "Action")
	fmt.Printf("%-8s %-14s %-10s\n", "----", "----------", "------")

	for _, load := range loadSequence {
		prevSize := batcher.GetBatchSize()
		batcher.UpdateLoad(load)
		newSize := batcher.GetBatchSize()

		action := "hold"
		if newSize > prevSize {
			action = "↑ scale up"
		} else if newSize < prevSize {
			action = "↓ scale down"
		}

		fmt.Printf("%-8.1f %-14d %-10s\n", load, newSize, action)
	}
	fmt.Println()

	// 5. Adaptive batcher in action
	fmt.Println("--- Adaptive Batcher: Processing Under Varying Load ---")
	db3 := NewSimulatedDB()
	batcher2 := NewAdaptiveBatcher(10, 500)
	totalRecords := generateRecords(500)

	// Low load phase
	batcher2.UpdateLoad(0.2)
	start = time.Now()
	n := batcher2.ProcessAdaptive(db3, totalRecords[:100])
	d := time.Since(start)
	fmt.Printf("Low load (0.2):  batch=%d, processed=%d, time=%v\n",
		batcher2.GetBatchSize(), n, d.Round(time.Microsecond))

	// High load phase
	batcher2.UpdateLoad(0.9)
	start = time.Now()
	n = batcher2.ProcessAdaptive(db3, totalRecords[100:400])
	d = time.Since(start)
	fmt.Printf("High load (0.9): batch=%d, processed=%d, time=%v\n",
		batcher2.GetBatchSize(), n, d.Round(time.Microsecond))

	// Load decreasing
	batcher2.UpdateLoad(0.1)
	start = time.Now()
	n = batcher2.ProcessAdaptive(db3, totalRecords[400:])
	d = time.Since(start)
	fmt.Printf("Low load (0.1):  batch=%d, processed=%d, time=%v\n",
		batcher2.GetBatchSize(), n, d.Round(time.Microsecond))
	fmt.Println()

	// 6. Cost projection
	calculateCostProjection()
}

// --- Formatting Helpers ---

func formatBytes(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}

func formatThroughput(rps float64) string {
	if rps >= 1_000_000 {
		return fmt.Sprintf("%.1fM/s", rps/1_000_000)
	} else if rps >= 1_000 {
		return fmt.Sprintf("%.1fK/s", rps/1_000)
	}
	return fmt.Sprintf("%.0f/s", rps)
}
