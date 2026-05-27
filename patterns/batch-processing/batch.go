package batch_processing

import (
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
