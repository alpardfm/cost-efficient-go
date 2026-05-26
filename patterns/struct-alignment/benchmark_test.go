package main

import (
	"testing"
	"unsafe"
)

// Global variable to prevent compiler optimizations
var (
	globalBadUsers  []BadUser
	globalGoodUsers []GoodUser
	globalInt       int
)

func Benchmark_BadUserAllocation(b *testing.B) {
	b.ReportAllocs() // Report memory allocations
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		users := make([]BadUser, 0, 1000)
		for j := 0; j < 1000; j++ {
			users = append(users, BadUser{
				ID:     int32(j),
				Active: j%2 == 0,
				Name:   "Test User",
				Age:    int8(j % 100),
			})
		}
		globalBadUsers = users // Prevent optimization
		globalInt = len(users)
	}
}

func Benchmark_GoodUserAllocation(b *testing.B) {
	b.ReportAllocs() // Report memory allocations
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		users := make([]GoodUser, 0, 1000)
		for j := 0; j < 1000; j++ {
			users = append(users, GoodUser{
				ID:     int32(j),
				Age:    int8(j % 100),
				Active: j%2 == 0,
				Name:   "Test User",
			})
		}
		globalGoodUsers = users // Prevent optimization
		globalInt = len(users)
	}
}

func Benchmark_BadUserWithPreAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pre-allocate exact capacity
		users := make([]BadUser, 1000)
		for j := 0; j < 1000; j++ {
			users[j] = BadUser{
				ID:     int32(j),
				Active: j%2 == 0,
				Name:   "Test User",
				Age:    int8(j % 100),
			}
		}
		globalBadUsers = users
	}
}

func Benchmark_GoodUserWithPreAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pre-allocate exact capacity
		users := make([]GoodUser, 1000)
		for j := 0; j < 1000; j++ {
			users[j] = GoodUser{
				ID:     int32(j),
				Age:    int8(j % 100),
				Active: j%2 == 0,
				Name:   "Test User",
			}
		}
		globalGoodUsers = users
	}
}

func Test_StructSizes(t *testing.T) {
	badSize := unsafe.Sizeof(BadUser{})
	goodSize := unsafe.Sizeof(GoodUser{})

	t.Logf("BadUser struct size:  %d bytes", badSize)
	t.Logf("GoodUser struct size: %d bytes", goodSize)
	t.Logf("Difference:           %d bytes", badSize-goodSize)
	t.Logf("Improvement:          %.1f%%", float64(badSize-goodSize)/float64(badSize)*100)

	// Verify optimization actually saves space
	if badSize <= goodSize {
		t.Errorf("Expected BadUser (%d) to be larger than GoodUser (%d)", badSize, goodSize)
	}

	// We expect at least 4 bytes savings (typical padding)
	if badSize-goodSize < 4 {
		t.Errorf("Expected at least 4 bytes savings, got %d", badSize-goodSize)
	}
}

func Test_MemoryAlignment(t *testing.T) {
	// Test field offsets
	var bad BadUser
	var good GoodUser

	badIDOffset := unsafe.Offsetof(bad.ID)
	badActiveOffset := unsafe.Offsetof(bad.Active)
	badNameOffset := unsafe.Offsetof(bad.Name)
	badAgeOffset := unsafe.Offsetof(bad.Age)

	goodIDOffset := unsafe.Offsetof(good.ID)
	goodAgeOffset := unsafe.Offsetof(good.Age)
	goodActiveOffset := unsafe.Offsetof(good.Active)
	goodNameOffset := unsafe.Offsetof(good.Name)

	t.Log("BadUser field offsets:")
	t.Logf("  ID:     %d", badIDOffset)
	t.Logf("  Active: %d", badActiveOffset)
	t.Logf("  Name:   %d", badNameOffset)
	t.Logf("  Age:    %d", badAgeOffset)

	t.Log("GoodUser field offsets:")
	t.Logf("  ID:     %d", goodIDOffset)
	t.Logf("  Age:    %d", goodAgeOffset)
	t.Logf("  Active: %d", goodActiveOffset)
	t.Logf("  Name:   %d", goodNameOffset)

	// Check for padding in BadUser
	if badActiveOffset != 4 {
		t.Errorf("Expected Active at offset 4, got %d", badActiveOffset)
	}
	if badNameOffset != 8 { // Should be padded to 8-byte boundary
		t.Errorf("Expected Name at offset 8 (8-byte aligned), got %d", badNameOffset)
	}
}
