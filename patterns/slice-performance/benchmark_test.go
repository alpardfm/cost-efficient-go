package main

import (
	"testing"
)

// Global variables to prevent compiler optimization
var (
	globalIntSlice []int
	globalInt      int
)

// ========== BASIC APPEND BENCHMARKS ==========

func Benchmark_NaiveAppend_100(b *testing.B) {
	benchmarkNaiveAppendHelper(b, 100)
}

func Benchmark_NaiveAppend_1000(b *testing.B) {
	benchmarkNaiveAppendHelper(b, 1000)
}

func Benchmark_NaiveAppend_10000(b *testing.B) {
	benchmarkNaiveAppendHelper(b, 10000)
}

func benchmarkNaiveAppendHelper(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var data []int
		for j := 0; j < size; j++ {
			data = append(data, j)
		}
		globalIntSlice = data
		globalInt = len(data)
	}
}

func Benchmark_MakeAppend_100(b *testing.B) {
	benchmarkMakeAppendHelper(b, 100)
}

func Benchmark_MakeAppend_1000(b *testing.B) {
	benchmarkMakeAppendHelper(b, 1000)
}

func Benchmark_MakeAppend_10000(b *testing.B) {
	benchmarkMakeAppendHelper(b, 10000)
}

func benchmarkMakeAppendHelper(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data := make([]int, 0, size)
		for j := 0; j < size; j++ {
			data = append(data, j)
		}
		globalIntSlice = data
		globalInt = len(data)
	}
}

func Benchmark_FixedArray_100(b *testing.B) {
	benchmarkFixedArrayHelper(b, 100)
}

func Benchmark_FixedArray_1000(b *testing.B) {
	benchmarkFixedArrayHelper(b, 1000)
}

func Benchmark_FixedArray_10000(b *testing.B) {
	benchmarkFixedArrayHelper(b, 10000)
}

func benchmarkFixedArrayHelper(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data := make([]int, size)
		for j := 0; j < size; j++ {
			data[j] = j
		}
		globalIntSlice = data
		globalInt = len(data)
	}
}

// ========== REAL-WORLD SCENARIOS ==========

type User struct {
	ID    int
	Name  string
	Email string
	Age   int
}

func Benchmark_ProcessUsers_Naive(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulating processing users from database
		var users []User

		// Simulate 1000 users from DB
		for j := 0; j < 1000; j++ {
			user := User{
				ID:    j,
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			}
			users = append(users, user)
		}

		globalInt = len(users)
	}
}

func Benchmark_ProcessUsers_Preallocated(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pre-allocate knowing we have 1000 users
		users := make([]User, 0, 1000)

		for j := 0; j < 1000; j++ {
			user := User{
				ID:    j,
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			}
			users = append(users, user)
		}

		globalInt = len(users)
	}
}

// ========== SLICE COPYING BENCHMARKS ==========

func Benchmark_SliceCopy_Append(b *testing.B) {
	b.ReportAllocs()

	src := make([]int, 1000)
	for i := range src {
		src[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dest := make([]int, 0, len(src))
		dest = append(dest, src...)
		globalIntSlice = dest
	}
}

func Benchmark_SliceCopy_MakeCopy(b *testing.B) {
	b.ReportAllocs()

	src := make([]int, 1000)
	for i := range src {
		src[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dest := make([]int, len(src))
		copy(dest, src)
		globalIntSlice = dest
	}
}

// ========== SLICE GROWTH PATTERN TESTS ==========

func Test_SliceGrowthPattern(t *testing.T) {
	// Test the growth algorithm
	var s []int

	expectedGrowth := []struct {
		appends int
		cap     int
	}{
		{1, 1},
		{2, 2},
		{3, 4},
		{5, 8},
		{9, 16},
		{17, 32},
		{33, 64},
		{1025, 1280}, // 1024 + 25%
	}

	for _, expected := range expectedGrowth {
		// Reset slice
		s = nil

		// Append expected number of times
		for i := 0; i < expected.appends; i++ {
			s = append(s, i)
		}

		if cap(s) != expected.cap {
			t.Errorf("After %d appends: expected cap=%d, got cap=%d",
				expected.appends, expected.cap, cap(s))
		} else {
			t.Logf("After %d appends: cap=%d (correct)", expected.appends, cap(s))
		}
	}
}

func Test_PreallocationSavings(t *testing.T) {
	// Demonstrate that pre-allocation saves allocations
	size := 1000

	// Count allocations for naive append
	allocCountNaive := 0
	var s1 []int
	for i := 0; i < size; i++ {
		oldCap := cap(s1)
		s1 = append(s1, i)
		if cap(s1) > oldCap {
			allocCountNaive++
		}
	}

	// Count allocations for pre-allocated
	allocCountPrealloc := 1 // The make() call
	s2 := make([]int, 0, size)
	for i := 0; i < size; i++ {
		s2 = append(s2, i)
		// No reallocations should happen
	}

	t.Logf("Naive append: %d allocations", allocCountNaive)
	t.Logf("Pre-allocated: %d allocations", allocCountPrealloc)
	t.Logf("Savings: %d fewer allocations (%.1f%%)",
		allocCountNaive-allocCountPrealloc,
		float64(allocCountNaive-allocCountPrealloc)/float64(allocCountNaive)*100)

	if allocCountNaive <= allocCountPrealloc {
		t.Error("Expected naive append to have more allocations than pre-allocated")
	}
}

// ========== MEMORY EFFICIENCY TEST ==========

func Test_MemoryEfficiency(t *testing.T) {
	// Test that pre-allocation uses memory more efficiently
	size := 1000

	// Naive approach
	var s1 []int
	for i := 0; i < size; i++ {
		s1 = append(s1, i)
	}

	// Pre-allocated approach
	s2 := make([]int, 0, size)
	for i := 0; i < size; i++ {
		s2 = append(s2, i)
	}

	// Calculate waste (capacity - length)
	waste1 := cap(s1) - len(s1)
	waste2 := cap(s2) - len(s2)

	t.Logf("Naive: capacity=%d, length=%d, waste=%d slots (%.1f%%)",
		cap(s1), len(s1), waste1, float64(waste1)/float64(cap(s1))*100)
	t.Logf("Pre-alloc: capacity=%d, length=%d, waste=%d slots (%.1f%%)",
		cap(s2), len(s2), waste2, float64(waste2)/float64(cap(s2))*100)

	if waste1 < waste2 {
		t.Error("Expected naive approach to have more wasted capacity")
	}
}
