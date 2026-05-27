// Package main demonstrates batch processing vs individual operations.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/batch-processing/examples/
package main

import (
	"fmt"
	"runtime"
	"time"

	batch "github.com/alpardfm/cost-efficient-go/patterns/batch-processing"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("PATTERN: Batch Processing vs Individual Operations")
	fmt.Println("============================================================")
	fmt.Println()

	// --- Individual vs Batch INSERT ---
	fmt.Println("--- Individual vs Batch INSERT (100 records) ---")
	const numRecords = 100

	records := make([]batch.Record, numRecords)
	for i := range records {
		records[i].ID = i + 1
	}

	// Individual INSERT
	db1 := batch.NewSimulatedDB()
	start := time.Now()
	batch.IndividualInsert(db1, records)
	individualTime := time.Since(start)
	fmt.Printf("Individual INSERT ×%d: %v (%.1f μs/record)\n",
		numRecords, individualTime, float64(individualTime.Microseconds())/float64(numRecords))

	// Batch INSERT (size=50)
	db2 := batch.NewSimulatedDB()
	start = time.Now()
	batch.BatchInsert(db2, records, 50)
	batchTime := time.Since(start)
	fmt.Printf("Batch INSERT (size=50) ×%d: %v (%.1f μs/record)\n",
		numRecords, batchTime, float64(batchTime.Microseconds())/float64(numRecords))

	speedup := float64(individualTime) / float64(batchTime)
	fmt.Printf("Speedup: %.1fx\n", speedup)
	fmt.Println()

	// --- Diminishing Returns ---
	fmt.Println("--- Diminishing Returns (1000 records, varying batch size) ---")
	fmt.Printf("%-12s %-15s %-15s %-12s\n", "Batch Size", "Total Time", "Per Record", "Throughput")
	fmt.Println("------------------------------------------------------------")

	batchSizes := []int{1, 10, 50, 100, 500, 1000}
	const dimRecords = 1000

	dimRecs := make([]batch.Record, dimRecords)
	for i := range dimRecs {
		dimRecs[i].ID = i + 1
	}

	for _, bs := range batchSizes {
		db := batch.NewSimulatedDB()
		start := time.Now()
		batch.BatchInsert(db, dimRecs, bs)
		elapsed := time.Since(start)

		perRecord := elapsed / time.Duration(dimRecords)
		throughput := float64(dimRecords) / elapsed.Seconds()
		fmt.Printf("%-12d %-15v %-15v %.0f/s\n", bs, elapsed, perRecord, throughput)
	}
	fmt.Println()

	// --- Adaptive Batcher ---
	fmt.Println("--- Adaptive Batcher ---")
	batcher := batch.NewAdaptiveBatcher(10, 1000)

	loads := []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1.0}
	for _, load := range loads {
		batcher.UpdateLoad(load)
		fmt.Printf("Load=%.1f → Batch Size=%d\n", load, batcher.GetBatchSize())
	}
	fmt.Println()

	// --- Memory Trade-off ---
	fmt.Println("--- Memory Trade-off ---")
	fmt.Printf("%-12s %-15s\n", "Batch Size", "Memory Used")
	fmt.Println("---------------------------")

	memBatchSizes := []int{1, 100, 1000, 10000}
	const memRecords = 10000

	memRecs := make([]batch.Record, memRecords)
	for i := range memRecs {
		memRecs[i].ID = i + 1
	}

	for _, bs := range memBatchSizes {
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		db := batch.NewSimulatedDB()
		batch.BatchInsert(db, memRecs, bs)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		memUsed := memAfter.TotalAlloc - memBefore.TotalAlloc
		fmt.Printf("%-12d %s\n", bs, formatBytes(memUsed))
	}
	fmt.Println()

	// --- Cost Projection ---
	fmt.Println("--- Cost Projection (10M ops/day) ---")
	const opsPerDay = 10_000_000
	const batchSize = 100
	individualTrips := opsPerDay
	batchTrips := opsPerDay / batchSize

	fmt.Printf("Individual: %d round-trips/day\n", individualTrips)
	fmt.Printf("Batch (size=%d): %d round-trips/day\n", batchSize, batchTrips)
	fmt.Printf("Reduction: %.1f%%\n", float64(individualTrips-batchTrips)/float64(individualTrips)*100)
}

func formatBytes(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}
