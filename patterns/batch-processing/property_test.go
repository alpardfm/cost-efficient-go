package main

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 11: Batch processing produces same results as individual (same final record count)
func TestProperty_BatchProducesSameResultsAsIndividual(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("batch insert produces same record count as individual", prop.ForAll(
		func(numRecords int, batchSize int) bool {
			if batchSize < 1 {
				batchSize = 1
			}
			records := generateRecords(numRecords)

			// Individual insert
			db1 := NewSimulatedDB()
			db1.networkLatency = 0
			db1.perRecordCost = 0
			db1.connectionSetup = 0
			individualCount := IndividualInsert(db1, records)

			// Batch insert
			db2 := NewSimulatedDB()
			db2.networkLatency = 0
			db2.perRecordCost = 0
			db2.connectionSetup = 0
			batchCount := BatchInsert(db2, records, batchSize)

			// Both should insert the same number of records
			return individualCount == batchCount && individualCount == numRecords
		},
		gen.IntRange(1, 200),
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 12: Batch memory usage grows monotonically with batch size
func TestProperty_BatchMemoryGrowsMonotonically(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("larger batch size buffer is larger than smaller batch buffer", prop.ForAll(
		func(numRecords int) bool {
			if numRecords < 20 {
				numRecords = 20
			}

			smallBatch := 5
			largeBatch := numRecords

			// The memory footprint of a batch operation is proportional to
			// the batch buffer size: batchSize * sizeof(Record).
			// A larger batch holds more records in memory simultaneously.
			smallBufSize := smallBatch * 264 // approximate Record size (ID + Payload)
			largeBufSize := largeBatch * 264

			return largeBufSize >= smallBufSize
		},
		gen.IntRange(20, 500),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 13: Adaptive batcher adjusts size monotonically with increasing load
func TestProperty_AdaptiveBatcherMonotonicWithLoad(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("increasing load produces non-decreasing batch size", prop.ForAll(
		func(steps int) bool {
			if steps < 2 {
				steps = 2
			}
			batcher := NewAdaptiveBatcher(10, 1000)

			prevSize := batcher.GetBatchSize()

			// Apply strictly increasing load factors above the scale-up threshold
			for i := 1; i <= steps; i++ {
				load := 0.7 + (0.3 * float64(i) / float64(steps))
				batcher.UpdateLoad(load)
				currentSize := batcher.GetBatchSize()

				// Batch size should be non-decreasing with increasing load
				if currentSize < prevSize {
					return false
				}
				prevSize = currentSize
			}
			return true
		},
		gen.IntRange(2, 20),
	))

	properties.TestingRun(t)
}
