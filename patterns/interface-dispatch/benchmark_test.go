package main

import (
	"testing"
)

// ============================================================
// BENCHMARK: Interface Dispatch vs Concrete Type
// ============================================================
//
// Requirements validated:
// - 4.1: Measure overhead of interface method call vs concrete type on hot path
// - 4.2: Demonstrate impact of interface on inlining decisions by Go compiler
// - 4.3: Measure ns/op AND allocations simultaneously on 1M+ iterations
// - 4.5: Cost projection — when does interface overhead matter at production scale
//
// Compiler Inlining Analysis:
//   Run: go build -gcflags="-m" ./patterns/interface-dispatch/
//
//   Expected observations:
//   - ConcreteProcessor.Process: "can inline" (simple arithmetic, small body)
//   - HotLoopConcrete: "inlining call to (*ConcreteProcessor).Process"
//   - HotLoopInterface: NO inlining message for Process call (interface dispatch)
//
//   Why: The Go compiler cannot inline through an interface because the concrete
//   type is unknown at compile time. This adds ~1-3ns per call from:
//   1. Indirect function call (vtable lookup via itab)
//   2. Lost inlining opportunity (no constant folding, no dead code elimination)
//
// To verify inlining decisions:
//   go build -gcflags="-m -m" ./patterns/interface-dispatch/ 2>&1 | grep -E "(can inline|inlining|devirtualize)"
// ============================================================

// Global variables to prevent compiler from optimizing away benchmark results.
var (
	globalFloat64 float64
)

// --- Setup helpers ---

func newTestProcessor() *ConcreteProcessor {
	return NewConcreteProcessor(2.5, 1.0)
}

// --- Benchmark: Concrete vs Interface on Hot Loop (1M iterations) ---

// BenchmarkHotLoop_Concrete measures direct method calls in a tight loop.
// The compiler CAN inline Process() here, resulting in faster execution.
func BenchmarkHotLoop_Concrete(b *testing.B) {
	p := newTestProcessor()
	const iterations = 1_000_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopConcrete(p, iterations)
	}
}

// BenchmarkHotLoop_Interface measures interface dispatch calls in a tight loop.
// The compiler CANNOT inline Process() here due to interface indirection.
func BenchmarkHotLoop_Interface(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	const iterations = 1_000_000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopInterface(p, iterations)
	}
}

// --- Benchmark: Single Method Call (Process) ---

// BenchmarkProcess_Concrete measures a single direct method call.
func BenchmarkProcess_Concrete(b *testing.B) {
	p := newTestProcessor()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.Process(float64(i))
	}
}

// BenchmarkProcess_Interface measures a single interface dispatch call.
func BenchmarkProcess_Interface(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.Process(float64(i))
	}
}

// --- Benchmark: Transform Method (more complex computation) ---

// BenchmarkTransform_Concrete measures Transform via concrete type.
func BenchmarkTransform_Concrete(b *testing.B) {
	p := newTestProcessor()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.Transform(float64(i))
	}
}

// BenchmarkTransform_Interface measures Transform via interface dispatch.
func BenchmarkTransform_Interface(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.Transform(float64(i))
	}
}

// --- Benchmark: BatchProcess (loop inside method) ---

// BenchmarkBatchProcess_Concrete measures BatchProcess via concrete type.
func BenchmarkBatchProcess_Concrete(b *testing.B) {
	p := newTestProcessor()
	values := make([]float64, 1_000_000)
	for i := range values {
		values[i] = float64(i) * 0.001
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.BatchProcess(values)
	}
}

// BenchmarkBatchProcess_Interface measures BatchProcess via interface dispatch.
// Note: The interface overhead here is only on the outer BatchProcess call.
// The internal loop in BatchProcess calls p.Process on concrete type (self-call).
func BenchmarkBatchProcess_Interface(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	values := make([]float64, 1_000_000)
	for i := range values {
		values[i] = float64(i) * 0.001
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = p.BatchProcess(values)
	}
}

// --- Benchmark: Boundary Pattern (interface at boundary, concrete internally) ---

// BenchmarkBoundaryPattern measures the recommended pattern:
// accept interface at boundary, use concrete type internally.
func BenchmarkBoundaryPattern(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	svc := NewService(p)
	values := make([]float64, 1_000_000)
	for i := range values {
		values[i] = float64(i) * 0.001
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		globalFloat64 = svc.ProcessBatch(values)
	}
}

// --- Benchmark: Varying Iteration Counts ---
// Shows how overhead scales with loop count.

func BenchmarkHotLoop_Concrete_10K(b *testing.B) {
	p := newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopConcrete(p, 10_000)
	}
}

func BenchmarkHotLoop_Interface_10K(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopInterface(p, 10_000)
	}
}

func BenchmarkHotLoop_Concrete_100K(b *testing.B) {
	p := newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopConcrete(p, 100_000)
	}
}

func BenchmarkHotLoop_Interface_100K(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopInterface(p, 100_000)
	}
}

func BenchmarkHotLoop_Concrete_10M(b *testing.B) {
	p := newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopConcrete(p, 10_000_000)
	}
}

func BenchmarkHotLoop_Interface_10M(b *testing.B) {
	var p ProcessorInterface = newTestProcessor()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFloat64 = HotLoopInterface(p, 10_000_000)
	}
}
