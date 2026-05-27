package main

import (
	"fmt"
	"math"
	"time"
)

// ============================================================
// PATTERN 14: Interface Dispatch vs Concrete Type
// ============================================================
// Problem: Interface method calls in Go prevent compiler inlining
// and add indirect call overhead. In tight loops (1M+ iterations),
// this overhead accumulates and becomes measurable.
//
// This pattern demonstrates:
// 1. Interface dispatch overhead in tight loops
// 2. How interfaces prevent compiler inlining
// 3. The "interface at boundary, concrete internally" pattern
// 4. Verdict: negligible for most code, only matters in tight loops
// ============================================================

// --- Concrete Type ---

// ConcreteProcessor performs data processing with direct method calls.
// The compiler can inline these methods when called on the concrete type.
type ConcreteProcessor struct {
	multiplier float64
	offset     float64
}

// NewConcreteProcessor creates a processor with given parameters.
func NewConcreteProcessor(multiplier, offset float64) *ConcreteProcessor {
	return &ConcreteProcessor{
		multiplier: multiplier,
		offset:     offset,
	}
}

// Process performs a computation on the input value.
// When called on concrete type, Go compiler can inline this.
func (p *ConcreteProcessor) Process(value float64) float64 {
	return value*p.multiplier + p.offset
}

// Transform applies a secondary transformation.
func (p *ConcreteProcessor) Transform(value float64) float64 {
	return math.Sqrt(math.Abs(value)) * p.multiplier
}

// BatchProcess processes a slice of values.
func (p *ConcreteProcessor) BatchProcess(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += p.Process(v)
	}
	return sum
}

// --- Interface Definition ---

// ProcessorInterface defines the contract for data processors.
// Calling methods through this interface adds indirect dispatch overhead
// and prevents the compiler from inlining the method body.
type ProcessorInterface interface {
	Process(value float64) float64
	Transform(value float64) float64
	BatchProcess(values []float64) float64
}

// --- Hot Loop Implementations ---

// HotLoopConcrete calls Process directly on the concrete type in a tight loop.
// The compiler CAN inline the method call here (after pattern / fast).
func HotLoopConcrete(p *ConcreteProcessor, iterations int) float64 {
	var result float64
	for i := range iterations {
		result += p.Process(float64(i))
	}
	return result
}

// HotLoopInterface calls Process through the interface in a tight loop.
// The compiler CANNOT inline the method call here (before pattern / slower).
func HotLoopInterface(p ProcessorInterface, iterations int) float64 {
	var result float64
	for i := range iterations {
		result += p.Process(float64(i))
	}
	return result
}

// --- Boundary Pattern ---

// Service demonstrates "interface at boundary, concrete internally".
// External callers interact through the interface (flexibility),
// but internal hot paths use the concrete type (performance).
type Service struct {
	// Internal: concrete type for hot path performance
	processor *ConcreteProcessor
}

// NewService creates a service — accepts interface at the boundary for testing/DI.
func NewService(p ProcessorInterface) *Service {
	// Type assert at boundary to get concrete type for internal use.
	// If not the expected type, wrap it.
	if concrete, ok := p.(*ConcreteProcessor); ok {
		return &Service{processor: concrete}
	}
	// Fallback: use a default concrete processor
	return &Service{processor: NewConcreteProcessor(1.0, 0.0)}
}

// ProcessBatch is the internal hot path — uses concrete type directly.
// No interface overhead in the tight loop.
func (s *Service) ProcessBatch(values []float64) float64 {
	return s.processor.BatchProcess(values)
}

// BoundaryPattern demonstrates the full pattern:
// - Interface at API boundary (clean architecture, testable)
// - Concrete type in internal hot loops (performance)
func BoundaryPattern() {
	fmt.Println("--- Boundary Pattern: Interface at Boundary, Concrete Internally ---")
	fmt.Println()

	// At the boundary: accept interface (flexible, testable)
	var p ProcessorInterface = NewConcreteProcessor(2.5, 1.0)

	// Create service — interface at boundary
	svc := NewService(p)

	// Internal hot path uses concrete type — no interface overhead
	values := make([]float64, 1_000_000)
	for i := range values {
		values[i] = float64(i) * 0.001
	}

	start := time.Now()
	result := svc.ProcessBatch(values)
	duration := time.Since(start)

	fmt.Printf("  Processed 1M values via concrete internal path\n")
	fmt.Printf("  Result: %.2f\n", result)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Println()
	fmt.Println("  Pattern summary:")
	fmt.Println("  ┌─────────────────────────────────────────────────────┐")
	fmt.Println("  │  API Boundary (interface)                           │")
	fmt.Println("  │    → Accepts ProcessorInterface for DI/testing      │")
	fmt.Println("  │                                                     │")
	fmt.Println("  │  Internal Hot Path (concrete)                       │")
	fmt.Println("  │    → Uses *ConcreteProcessor directly               │")
	fmt.Println("  │    → Compiler can inline Process() calls            │")
	fmt.Println("  │    → Zero interface dispatch overhead               │")
	fmt.Println("  └─────────────────────────────────────────────────────┘")
	fmt.Println()
}

// --- Cost Projection ---

// CostProjection calculates when interface overhead matters at scale.
func CostProjection() {
	fmt.Println("--- Cost Projection: When Does Interface Overhead Matter? ---")
	fmt.Println()

	// Typical overhead per interface call: ~1-3ns (vs inlined concrete: ~0.3-0.5ns)
	// This is the indirect call + inability to inline
	interfaceOverheadNs := 2.0 // Conservative estimate per call
	concreteNs := 0.4          // Inlined concrete call

	scales := []struct {
		name       string
		opsPerDay  int64
		callsPerOp int
	}{
		{"REST API handler (few calls per request)", 10_000_000, 5},
		{"Stream processor (moderate loop)", 10_000_000, 100},
		{"Tight computation loop (hot path)", 10_000_000, 1_000_000},
	}

	costPerVCPUHour := 0.0416 // t3.medium

	fmt.Printf("  %-50s %12s %12s %10s\n", "Scenario", "Interface", "Concrete", "Savings")
	fmt.Printf("  %-50s %12s %12s %10s\n", "--------", "---------", "--------", "-------")

	for _, s := range scales {
		totalCalls := s.opsPerDay * int64(s.callsPerOp)

		interfaceTimeS := float64(totalCalls) * interfaceOverheadNs / 1e9
		concreteTimeS := float64(totalCalls) * concreteNs / 1e9

		interfaceCostDay := (interfaceTimeS / 3600) * costPerVCPUHour
		concreteCostDay := (concreteTimeS / 3600) * costPerVCPUHour
		savingsDay := interfaceCostDay - concreteCostDay

		fmt.Printf("  %-50s $%10.6f $%10.6f $%8.6f\n",
			s.name, interfaceCostDay, concreteCostDay, savingsDay)
	}

	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  VERDICT                                                        │")
	fmt.Println("  │                                                                 │")
	fmt.Println("  │  • REST APIs (5 calls/request): Interface overhead is NOISE     │")
	fmt.Println("  │    → $0.000001/day — literally unmeasurable                     │")
	fmt.Println("  │                                                                 │")
	fmt.Println("  │  • Stream processors (100 calls/op): Still negligible           │")
	fmt.Println("  │    → Use interfaces freely for clean architecture               │")
	fmt.Println("  │                                                                 │")
	fmt.Println("  │  • Tight loops (1M+ calls/op): Overhead becomes measurable      │")
	fmt.Println("  │    → Use concrete types in hot inner loops                      │")
	fmt.Println("  │    → Keep interfaces at boundaries only                         │")
	fmt.Println("  │                                                                 │")
	fmt.Println("  │  Rule of thumb: If your loop runs < 10K iterations,             │")
	fmt.Println("  │  interface overhead does NOT matter. Use interfaces freely.     │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

// --- Main Demo ---

func main() {
	fmt.Println("=== Pattern 14: Interface Dispatch vs Concrete Type ===")
	fmt.Println()

	processor := NewConcreteProcessor(2.5, 1.0)
	iterations := 10_000_000

	// 1. Concrete type hot loop (fast — compiler can inline)
	fmt.Println("--- Hot Loop: Concrete Type (compiler can inline) ---")
	start := time.Now()
	resultConcrete := HotLoopConcrete(processor, iterations)
	concreteDuration := time.Since(start)
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Result: %.2f\n", resultConcrete)
	fmt.Printf("  Duration: %v\n", concreteDuration)
	fmt.Println()

	// 2. Interface hot loop (slower — indirect dispatch, no inlining)
	fmt.Println("--- Hot Loop: Interface Dispatch (no inlining possible) ---")
	var iface ProcessorInterface = processor
	start = time.Now()
	resultInterface := HotLoopInterface(iface, iterations)
	interfaceDuration := time.Since(start)
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Result: %.2f\n", resultInterface)
	fmt.Printf("  Duration: %v\n", interfaceDuration)
	fmt.Println()

	// 3. Comparison
	fmt.Println("--- Comparison ---")
	fmt.Printf("  Concrete:  %v\n", concreteDuration)
	fmt.Printf("  Interface: %v\n", interfaceDuration)
	if interfaceDuration > concreteDuration {
		overhead := float64(interfaceDuration-concreteDuration) / float64(concreteDuration) * 100
		fmt.Printf("  Overhead:  %.1f%% slower via interface\n", overhead)
		nsPerCallConcrete := float64(concreteDuration.Nanoseconds()) / float64(iterations)
		nsPerCallInterface := float64(interfaceDuration.Nanoseconds()) / float64(iterations)
		fmt.Printf("  Per call:  %.2f ns (concrete) vs %.2f ns (interface)\n",
			nsPerCallConcrete, nsPerCallInterface)
	}
	fmt.Println()

	// 4. Boundary pattern demonstration
	BoundaryPattern()

	// 5. Cost projection
	CostProjection()

	// 6. Inlining note
	fmt.Println("--- Compiler Inlining Analysis ---")
	fmt.Println("  Run: go build -gcflags='-m' ./patterns/interface-dispatch/")
	fmt.Println()
	fmt.Println("  Expected output:")
	fmt.Println("    • ConcreteProcessor.Process: 'can inline'")
	fmt.Println("    • HotLoopConcrete: 'inlining call to ConcreteProcessor.Process'")
	fmt.Println("    • HotLoopInterface: NO inlining (interface dispatch)")
	fmt.Println()
	fmt.Println("  This is the key difference: the compiler cannot see through")
	fmt.Println("  the interface to inline the method body, adding ~1-3ns per call.")
	fmt.Println()
}
