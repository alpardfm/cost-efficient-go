package main

import (
	"fmt"
	"testing"
	"unsafe"
)

// Global variables to prevent optimization
type Entry struct {
	Key   int
	Value string
}

var (
	globalMap   map[int]string
	globalSlice []Entry
	globalInt   int
)

// ========== MAP VS SLICE BENCHMARKS ==========

func Benchmark_MapInsert_100(b *testing.B) {
	benchmarkMapInsert(b, 100)
}

func Benchmark_MapInsert_1000(b *testing.B) {
	benchmarkMapInsert(b, 1000)
}

func Benchmark_MapInsert_10000(b *testing.B) {
	benchmarkMapInsert(b, 10000)
}

func benchmarkMapInsert(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := make(map[int]string)
		for j := 0; j < size; j++ {
			m[j] = "value"
		}
		globalMap = m
		globalInt = len(m)
	}
}

func Benchmark_MapInsertPrealloc_100(b *testing.B) {
	benchmarkMapInsertPrealloc(b, 100)
}

func Benchmark_MapInsertPrealloc_1000(b *testing.B) {
	benchmarkMapInsertPrealloc(b, 1000)
}

func Benchmark_MapInsertPrealloc_10000(b *testing.B) {
	benchmarkMapInsertPrealloc(b, 10000)
}

func benchmarkMapInsertPrealloc(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := make(map[int]string, size) // Pre-allocate!
		for j := 0; j < size; j++ {
			m[j] = "value"
		}
		globalMap = m
		globalInt = len(m)
	}
}

func Benchmark_SliceStructInsert_100(b *testing.B) {
	benchmarkSliceStructInsert(b, 100)
}

func Benchmark_SliceStructInsert_1000(b *testing.B) {
	benchmarkSliceStructInsert(b, 1000)
}

func Benchmark_SliceStructInsert_10000(b *testing.B) {
	benchmarkSliceStructInsert(b, 10000)
}

func benchmarkSliceStructInsert(b *testing.B, size int) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := make([]Entry, 0, size)
		for j := 0; j < size; j++ {
			slice = append(slice, Entry{Key: j, Value: "value"})
		}
		globalSlice = slice
		globalInt = len(slice)
	}
}

// ========== LOOKUP BENCHMARKS ==========

func Benchmark_MapLookup(b *testing.B) {
	// Prepare map with 1000 entries
	m := make(map[int]string, 1000)
	for i := 0; i < 1000; i++ {
		m[i] = "value"
	}

	b.ReportAllocs()
	b.ResetTimer()

	var found string
	for i := 0; i < b.N; i++ {
		// Lookup random keys (but same key to be fair)
		found = m[i%1000]
	}
	_ = found
}

func Benchmark_SliceLookupBinarySearch(b *testing.B) {
	// Prepare sorted slice
	type entry struct {
		Key   int
		Value string
	}
	slice := make([]entry, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = entry{Key: i, Value: "value"}
	}

	b.ReportAllocs()
	b.ResetTimer()

	var found string
	for i := 0; i < b.N; i++ {
		// Binary search
		key := i % 1000
		low, high := 0, len(slice)-1
		for low <= high {
			mid := (low + high) / 2
			if slice[mid].Key == key {
				found = slice[mid].Value
				break
			} else if slice[mid].Key < key {
				low = mid + 1
			} else {
				high = mid - 1
			}
		}
	}
	_ = found
}

func Benchmark_SliceLookupDirect(b *testing.B) {
	// Prepare slice where index = key
	slice := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = "value"
	}

	b.ReportAllocs()
	b.ResetTimer()

	var found string
	for i := 0; i < b.N; i++ {
		found = slice[i%1000]
	}
	_ = found
}

// ========== ITERATION BENCHMARKS ==========

func Benchmark_MapIteration(b *testing.B) {
	m := make(map[int]string, 1000)
	for i := 0; i < 1000; i++ {
		m[i] = "value"
	}

	b.ReportAllocs()
	b.ResetTimer()

	var total int
	for i := 0; i < b.N; i++ {
		for k, v := range m {
			total += k
			_ = v
		}
	}
	globalInt = total
}

func Benchmark_SliceIteration(b *testing.B) {
	type entry struct {
		Key   int
		Value string
	}
	slice := make([]entry, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = entry{Key: i, Value: "value"}
	}

	b.ReportAllocs()
	b.ResetTimer()

	var total int
	for i := 0; i < b.N; i++ {
		for _, e := range slice {
			total += e.Key
			_ = e.Value
		}
	}
	globalInt = total
}

// ========== MEMORY OVERHEAD TESTS ==========

func Test_MapMemoryOverhead(t *testing.T) {
	// This test shows map overhead visually
	t.Log("Map memory overhead analysis:")
	t.Log("==============================")

	sizes := []int{10, 100, 1000, 10000}
	for _, size := range sizes {
		// Map
		m := make(map[int]string, size)
		for i := 0; i < size; i++ {
			m[i] = "x" // Single character value
		}

		// Slice of structs
		type entry struct {
			Key   int
			Value string
		}
		slice := make([]entry, 0, size)
		for i := 0; i < size; i++ {
			slice = append(slice, entry{Key: i, Value: "x"})
		}

		mapSize := int(unsafe.Sizeof(m)) // This underestimates, but shows pointer size
		sliceSize := len(slice) * int(unsafe.Sizeof(entry{}))

		t.Logf("Size %6d: Map pointer=%4d bytes, Slice=%6d bytes",
			size, mapSize, sliceSize)
		t.Logf("          Note: Real map overhead is ~50 bytes per entry!")
	}
}

func Test_PreallocationImpact(t *testing.T) {
	// Show that map pre-allocation matters
	size := 1000

	// Without pre-allocation
	alloc1 := testing.AllocsPerRun(100, func() {
		m := make(map[int]string)
		for i := 0; i < size; i++ {
			m[i] = "value"
		}
	})

	// With pre-allocation
	alloc2 := testing.AllocsPerRun(100, func() {
		m := make(map[int]string, size)
		for i := 0; i < size; i++ {
			m[i] = "value"
		}
	})

	t.Logf("Allocations for inserting %d entries:", size)
	t.Logf("  Without pre-allocation: %.1f allocations", alloc1)
	t.Logf("  With pre-allocation:    %.1f allocations", alloc2)
	t.Logf("  Improvement:            %.1f%% fewer allocations",
		(alloc1-alloc2)/alloc1*100)

	// Maps still allocate more than 1 due to internal structures
	if alloc2 >= alloc1 {
		t.Error("Expected pre-allocation to reduce allocations")
	}
}

func Test_MapVsSet(t *testing.T) {
	// Compare map[T]bool vs map[T]struct{} for sets
	size := 1000

	// map[string]bool
	mem1 := testing.AllocsPerRun(100, func() {
		set := make(map[string]bool)
		for i := 0; i < size; i++ {
			set[fmt.Sprintf("key_%d", i)] = true
		}
	})

	// map[string]struct{}
	mem2 := testing.AllocsPerRun(100, func() {
		set := make(map[string]struct{})
		for i := 0; i < size; i++ {
			set[fmt.Sprintf("key_%d", i)] = struct{}{}
		}
	})

	t.Logf("Set implementations comparison (%d elements):", size)
	t.Logf("  map[string]bool:      %.1f allocations", mem1)
	t.Logf("  map[string]struct{}:  %.1f allocations", mem2)
	t.Logf("  struct{} saves:       %.1f%% memory", (mem1-mem2)/mem1*100)

	// Note: Actual memory savings are bigger than allocation count suggests
	// because struct{} is 0 bytes vs bool which is at least 1 byte
}
