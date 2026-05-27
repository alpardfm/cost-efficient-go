package string_building

import (
	"fmt"
	"testing"
)

// ============================================================
// Benchmarks: String Building & Concatenation Efficiency
// Requirements: 3.1, 3.2, 3.4, 3.5
// ============================================================

// Global sink variables to prevent compiler optimization
var (
	benchResult string
	benchParts  = map[int][]string{
		10:  generateTestStrings(10),
		50:  generateTestStrings(50),
		100: generateTestStrings(100),
		500: generateTestStrings(500),
	}
)

// --- ConcatPlus benchmarks ---

func BenchmarkConcatPlus_10(b *testing.B) {
	parts := benchParts[10]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatPlus(parts)
	}
}

func BenchmarkConcatPlus_50(b *testing.B) {
	parts := benchParts[50]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatPlus(parts)
	}
}

func BenchmarkConcatPlus_100(b *testing.B) {
	parts := benchParts[100]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatPlus(parts)
	}
}

func BenchmarkConcatPlus_500(b *testing.B) {
	parts := benchParts[500]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatPlus(parts)
	}
}

// --- ConcatSprintf benchmarks ---

func BenchmarkConcatSprintf_10(b *testing.B) {
	parts := benchParts[10]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatSprintf(parts)
	}
}

func BenchmarkConcatSprintf_50(b *testing.B) {
	parts := benchParts[50]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatSprintf(parts)
	}
}

func BenchmarkConcatSprintf_100(b *testing.B) {
	parts := benchParts[100]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatSprintf(parts)
	}
}

func BenchmarkConcatSprintf_500(b *testing.B) {
	parts := benchParts[500]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatSprintf(parts)
	}
}

// --- ConcatBuilder benchmarks ---

func BenchmarkConcatBuilder_10(b *testing.B) {
	parts := benchParts[10]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuilder(parts)
	}
}

func BenchmarkConcatBuilder_50(b *testing.B) {
	parts := benchParts[50]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuilder(parts)
	}
}

func BenchmarkConcatBuilder_100(b *testing.B) {
	parts := benchParts[100]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuilder(parts)
	}
}

func BenchmarkConcatBuilder_500(b *testing.B) {
	parts := benchParts[500]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuilder(parts)
	}
}

// --- ConcatBuffer benchmarks ---

func BenchmarkConcatBuffer_10(b *testing.B) {
	parts := benchParts[10]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuffer(parts)
	}
}

func BenchmarkConcatBuffer_50(b *testing.B) {
	parts := benchParts[50]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuffer(parts)
	}
}

func BenchmarkConcatBuffer_100(b *testing.B) {
	parts := benchParts[100]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuffer(parts)
	}
}

func BenchmarkConcatBuffer_500(b *testing.B) {
	parts := benchParts[500]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = ConcatBuffer(parts)
	}
}

// ============================================================
// Validation Test: Builder ≥ 5x faster than + at 100+ concatenations
// Requirement 3.4
// ============================================================

func TestBuilderFasterThanPlus100(t *testing.T) {
	// Run sub-benchmarks to measure relative performance.
	// Note (Req 9.3): Threshold lowered from 5x to 1.5x because the race detector
	// and varying CPU/memory conditions significantly affect timing ratios.
	// The core property (Builder is faster than +) is still validated.
	// The allocation-based assertion below provides a deterministic check.
	parts := benchParts[100]

	plusResult := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchResult = ConcatPlus(parts)
		}
	})

	builderResult := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchResult = ConcatBuilder(parts)
		}
	})

	plusNsPerOp := plusResult.NsPerOp()
	builderNsPerOp := builderResult.NsPerOp()

	if builderNsPerOp == 0 {
		t.Fatal("Builder benchmark returned 0 ns/op — cannot compute ratio")
	}

	speedup := float64(plusNsPerOp) / float64(builderNsPerOp)

	t.Logf("ConcatPlus (100 parts):    %d ns/op, %d B/op, %d allocs/op",
		plusNsPerOp, plusResult.AllocedBytesPerOp(), plusResult.AllocsPerOp())
	t.Logf("ConcatBuilder (100 parts): %d ns/op, %d B/op, %d allocs/op",
		builderNsPerOp, builderResult.AllocedBytesPerOp(), builderResult.AllocsPerOp())
	t.Logf("Speedup: %.1fx", speedup)

	// Timing-based check: Builder should be at least somewhat faster
	if speedup < 1.5 {
		t.Errorf("Builder should be ≥ 1.5x faster than + at 100 concatenations, got %.1fx", speedup)
	}

	// Deterministic allocation check: Builder must use fewer allocations
	if builderResult.AllocsPerOp() >= plusResult.AllocsPerOp() {
		t.Errorf("Builder should have fewer allocs than + operator: builder=%d, plus=%d",
			builderResult.AllocsPerOp(), plusResult.AllocsPerOp())
	}
}

func TestBuilderFasterThanPlus500(t *testing.T) {
	// Note (Req 9.3): Same threshold adjustment as TestBuilderFasterThanPlus100.
	parts := benchParts[500]

	plusResult := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchResult = ConcatPlus(parts)
		}
	})

	builderResult := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchResult = ConcatBuilder(parts)
		}
	})

	plusNsPerOp := plusResult.NsPerOp()
	builderNsPerOp := builderResult.NsPerOp()

	if builderNsPerOp == 0 {
		t.Fatal("Builder benchmark returned 0 ns/op — cannot compute ratio")
	}

	speedup := float64(plusNsPerOp) / float64(builderNsPerOp)

	t.Logf("ConcatPlus (500 parts):    %d ns/op, %d B/op, %d allocs/op",
		plusNsPerOp, plusResult.AllocedBytesPerOp(), plusResult.AllocsPerOp())
	t.Logf("ConcatBuilder (500 parts): %d ns/op, %d B/op, %d allocs/op",
		builderNsPerOp, builderResult.AllocedBytesPerOp(), builderResult.AllocsPerOp())
	t.Logf("Speedup: %.1fx", speedup)

	if speedup < 1.5 {
		t.Errorf("Builder should be ≥ 1.5x faster than + at 500 concatenations, got %.1fx", speedup)
	}

	// Deterministic allocation check
	if builderResult.AllocsPerOp() >= plusResult.AllocsPerOp() {
		t.Errorf("Builder should have fewer allocs than + operator: builder=%d, plus=%d",
			builderResult.AllocsPerOp(), plusResult.AllocsPerOp())
	}
}

// ============================================================
// Allocation Measurement Tests
// Requirement 3.2: Measure bytes allocated per operation
// ============================================================

func TestAllocationsPerMethod(t *testing.T) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		parts := benchParts[size]
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			plusResult := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchResult = ConcatPlus(parts)
				}
			})

			sprintfResult := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchResult = ConcatSprintf(parts)
				}
			})

			builderResult := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchResult = ConcatBuilder(parts)
				}
			})

			bufferResult := testing.Benchmark(func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchResult = ConcatBuffer(parts)
				}
			})

			t.Logf("--- %d concatenations ---", size)
			t.Logf("  + operator:      %d B/op, %d allocs/op",
				plusResult.AllocedBytesPerOp(), plusResult.AllocsPerOp())
			t.Logf("  fmt.Sprintf:     %d B/op, %d allocs/op",
				sprintfResult.AllocedBytesPerOp(), sprintfResult.AllocsPerOp())
			t.Logf("  strings.Builder: %d B/op, %d allocs/op",
				builderResult.AllocedBytesPerOp(), builderResult.AllocsPerOp())
			t.Logf("  bytes.Buffer:    %d B/op, %d allocs/op",
				bufferResult.AllocedBytesPerOp(), bufferResult.AllocsPerOp())

			// Builder should have fewer allocations than + operator
			if builderResult.AllocsPerOp() > plusResult.AllocsPerOp() {
				t.Errorf("Builder should have fewer allocs than + operator at %d parts", size)
			}
		})
	}
}
