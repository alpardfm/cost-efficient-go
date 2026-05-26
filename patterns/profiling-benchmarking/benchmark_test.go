package main

import (
	"math/rand"
	"testing"
)

// ============================================================
// Benchmarks: Demonstrating correct benchmarking patterns
// ============================================================

// --- Common Pitfall: Compiler Elimination ---

// ❌ BAD: Compiler may eliminate the entire computation
func BenchmarkSortBad(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = rand.Intn(100000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortInts(data) // Result unused — compiler might optimize away
	}
}

// ✅ GOOD: Use package-level sink to prevent elimination
var benchSink interface{}

func BenchmarkSortGood(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = rand.Intn(100000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = SortInts(data)
	}
}

// --- Allocation Measurement ---

func BenchmarkAllocate1KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = AllocateSlices(100, 1024)
	}
}

func BenchmarkAllocate4KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = AllocateSlices(100, 4096)
	}
}

func BenchmarkAllocate64KB(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = AllocateSlices(100, 65536)
	}
}

// --- Scale Comparison ---

func BenchmarkProcessBatch100(b *testing.B) {
	data := make([]int, 100)
	for i := range data {
		data[i] = rand.Intn(1000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = ProcessBatch(data)
	}
}

func BenchmarkProcessBatch10K(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = rand.Intn(1000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = ProcessBatch(data)
	}
}

func BenchmarkProcessBatch1M(b *testing.B) {
	data := make([]int, 1000000)
	for i := range data {
		data[i] = rand.Intn(1000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = ProcessBatch(data)
	}
}

// --- Sort at Different Scales ---

func BenchmarkSort100(b *testing.B) {
	data := make([]int, 100)
	for i := range data {
		data[i] = rand.Intn(10000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SortInts(data)
	}
}

func BenchmarkSort10K(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = rand.Intn(100000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SortInts(data)
	}
}

func BenchmarkSort100K(b *testing.B) {
	data := make([]int, 100000)
	for i := range data {
		data[i] = rand.Intn(1000000)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchSink = SortInts(data)
	}
}
