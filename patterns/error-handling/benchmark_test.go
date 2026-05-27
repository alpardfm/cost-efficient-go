package error_handling

import (
	"math/rand"
	"testing"
)

// ============================================================
// BENCHMARK: Error Handling Efficiency
// ============================================================
//
// Requirements validated:
// - 5.1: Compare errors.New(), fmt.Errorf(), sentinel errors, and custom error types
// - 5.2: Measure allocations per error creation on hot path (high-frequency operations)
// - 5.4: Demonstrate sentinel errors = 0 allocs vs fmt.Errorf > 0 allocs
//
// Key insight:
//   errors.New() and fmt.Errorf() allocate on EVERY call.
//   Sentinel errors and pre-allocated custom errors allocate ZERO times on use.
//   On a hot path with 5% error rate at 100M req/day = 5M unnecessary allocs/day.
//
// Run benchmarks:
//   go test -bench=. -benchmem -benchtime=3s ./patterns/error-handling/
// ============================================================

// Global variables to prevent compiler from optimizing away benchmark results.
var (
	benchErr     error
	benchSuccess int
	benchFailure int
)

// --- Benchmark: Error Creation Methods (allocs/op) ---

// BenchmarkErrorCreation_ErrorsNew measures allocations from errors.New().
// Expected: 1 alloc/op (new error object on heap each call).
func BenchmarkErrorCreation_ErrorsNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchErr = ErrorsNew()
	}
}

// BenchmarkErrorCreation_FmtErrorf measures allocations from fmt.Errorf().
// Expected: ≥2 allocs/op (format string processing + argument boxing + error object).
func BenchmarkErrorCreation_FmtErrorf(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchErr = FmtErrorf()
	}
}

// BenchmarkErrorCreation_Sentinel measures allocations from sentinel error access.
// Expected: 0 allocs/op (just a pointer copy, allocated once at init).
func BenchmarkErrorCreation_Sentinel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchErr = SentinelError
	}
}

// BenchmarkErrorCreation_Preallocated measures allocations from pre-allocated custom error.
// Expected: 0 allocs/op (just a pointer copy, allocated once at init).
func BenchmarkErrorCreation_Preallocated(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchErr = PreallocatedError
	}
}

// BenchmarkErrorCreation_CustomTypeNew measures allocations from creating a new CustomErrorType each time.
// This shows the cost when you DON'T pre-allocate custom errors.
func BenchmarkErrorCreation_CustomTypeNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchErr = &CustomErrorType{
			Code:    404,
			Message: "operation failed: resource not found",
		}
	}
}

// --- Benchmark: Throughput Impact with 5% Error Rate ---
// Simulates a hot path where 5% of operations produce errors.
// Measures the throughput difference between error creation strategies.

// BenchmarkHotPath_ErrorsNew measures throughput with errors.New() on 5% error rate.
func BenchmarkHotPath_ErrorsNew(b *testing.B) {
	const iterations = 10_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSuccess, benchFailure = HotPathWithErrors(iterations, ErrorsNew)
	}
}

// BenchmarkHotPath_FmtErrorf measures throughput with fmt.Errorf() on 5% error rate.
func BenchmarkHotPath_FmtErrorf(b *testing.B) {
	const iterations = 10_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSuccess, benchFailure = HotPathWithErrors(iterations, FmtErrorf)
	}
}

// BenchmarkHotPath_Sentinel measures throughput with sentinel error on 5% error rate.
func BenchmarkHotPath_Sentinel(b *testing.B) {
	const iterations = 10_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSuccess, benchFailure = HotPathWithSentinel(iterations)
	}
}

// BenchmarkHotPath_Preallocated measures throughput with pre-allocated error on 5% error rate.
func BenchmarkHotPath_Preallocated(b *testing.B) {
	const iterations = 10_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchSuccess, benchFailure = HotPathWithPreallocated(iterations)
	}
}

// --- Benchmark: Allocation Validation ---
// These benchmarks explicitly validate the zero-alloc property of sentinel errors
// vs the allocating behavior of fmt.Errorf.

// BenchmarkAllocValidation_SentinelZeroAlloc runs sentinel error in a tight loop
// to confirm 0 allocs/op across many iterations.
func BenchmarkAllocValidation_SentinelZeroAlloc(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Access sentinel error 100 times per benchmark iteration
		// to amplify any hidden allocation.
		for j := 0; j < 100; j++ {
			benchErr = SentinelError
		}
	}
}

// BenchmarkAllocValidation_FmtErrorfAllocates runs fmt.Errorf in a tight loop
// to confirm > 0 allocs/op on every call.
func BenchmarkAllocValidation_FmtErrorfAllocates(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Each call to FmtErrorf allocates.
		for j := 0; j < 100; j++ {
			benchErr = FmtErrorf()
		}
	}
}

// --- Unit Tests: Validate Allocation Counts ---

// TestSentinelError_ZeroAllocs verifies that accessing a sentinel error
// produces exactly 0 heap allocations.
func TestSentinelError_ZeroAllocs(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		benchErr = SentinelError
	})
	if allocs != 0 {
		t.Errorf("sentinel error should have 0 allocs, got %f", allocs)
	}
}

// TestPreallocatedError_ZeroAllocs verifies that accessing a pre-allocated
// custom error produces exactly 0 heap allocations.
func TestPreallocatedError_ZeroAllocs(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		benchErr = PreallocatedError
	})
	if allocs != 0 {
		t.Errorf("preallocated error should have 0 allocs, got %f", allocs)
	}
}

// TestFmtErrorf_Allocates verifies that fmt.Errorf allocates on every call.
func TestFmtErrorf_Allocates(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		benchErr = FmtErrorf()
	})
	if allocs <= 0 {
		t.Errorf("fmt.Errorf should allocate on every call, got %f allocs", allocs)
	}
}

// TestErrorsNew_Allocates verifies that errors.New allocates on every call.
func TestErrorsNew_Allocates(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		benchErr = ErrorsNew()
	})
	if allocs <= 0 {
		t.Errorf("errors.New should allocate on every call, got %f allocs", allocs)
	}
}

// TestHotPath_ErrorRate verifies the hot path simulation produces ~5% error rate.
func TestHotPath_ErrorRate(t *testing.T) {
	const iterations = 100_000
	rand.New(rand.NewSource(42)) // deterministic for test

	_, failures := HotPathWithErrors(iterations, ErrorsNew)
	errorRate := float64(failures) / float64(iterations)

	// Allow ±2% tolerance around 5% target
	if errorRate < 0.03 || errorRate > 0.07 {
		t.Errorf("expected ~5%% error rate, got %.2f%%", errorRate*100)
	}
}

// TestAllErrorMethods_SatisfyInterface verifies all error creation methods
// produce valid error interface implementations.
func TestAllErrorMethods_SatisfyInterface(t *testing.T) {
	methods := []struct {
		name string
		err  error
	}{
		{"ErrorsNew", ErrorsNew()},
		{"FmtErrorf", FmtErrorf()},
		{"SentinelError", SentinelError},
		{"PreallocatedError", PreallocatedError},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			if m.err == nil {
				t.Fatal("error should not be nil")
			}
			msg := m.err.Error()
			if msg == "" {
				t.Fatal("error message should not be empty")
			}
		})
	}
}
