// Package main demonstrates interface dispatch vs concrete type patterns.
// This is the educational example code showing the cost-efficiency pattern.
//
// Run with: go run ./patterns/interface-dispatch/examples/
package main

import (
	"fmt"
	"time"

	iface "github.com/alpardfm/cost-efficient-go/patterns/interface-dispatch"
)

func main() {
	fmt.Println("=== Pattern 14: Interface Dispatch vs Concrete Type ===")
	fmt.Println()

	processor := iface.NewConcreteProcessor(2.5, 1.0)
	iterations := 10_000_000

	// 1. Concrete type hot loop (fast — compiler can inline)
	fmt.Println("--- Hot Loop: Concrete Type (compiler can inline) ---")
	start := time.Now()
	resultConcrete := iface.HotLoopConcrete(processor, iterations)
	concreteDuration := time.Since(start)
	fmt.Printf("  Iterations: %d\n", iterations)
	fmt.Printf("  Result: %.2f\n", resultConcrete)
	fmt.Printf("  Duration: %v\n", concreteDuration)
	fmt.Println()

	// 2. Interface hot loop (slower — indirect dispatch, no inlining)
	fmt.Println("--- Hot Loop: Interface Dispatch (no inlining possible) ---")
	var ifaceProc iface.ProcessorInterface = processor
	start = time.Now()
	resultInterface := iface.HotLoopInterface(ifaceProc, iterations)
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

// BoundaryPattern demonstrates the full pattern:
// - Interface at API boundary (clean architecture, testable)
// - Concrete type in internal hot loops (performance)
func BoundaryPattern() {
	fmt.Println("--- Boundary Pattern: Interface at Boundary, Concrete Internally ---")
	fmt.Println()

	// At the boundary: accept interface (flexible, testable)
	var p iface.ProcessorInterface = iface.NewConcreteProcessor(2.5, 1.0)

	// Create service — interface at boundary
	svc := iface.NewService(p)

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

// CostProjection calculates when interface overhead matters at scale.
func CostProjection() {
	fmt.Println("--- Cost Projection: When Does Interface Overhead Matter? ---")
	fmt.Println()

	costPerVCPUHour := 0.0416 // t3.medium

	scenarios := []iface.CostProjectionScenario{
		{Name: "REST API handler (few calls per request)", OpsPerDay: 10_000_000, CallsPerOp: 5},
		{Name: "Stream processor (moderate loop)", OpsPerDay: 10_000_000, CallsPerOp: 100},
		{Name: "Tight computation loop (hot path)", OpsPerDay: 10_000_000, CallsPerOp: 1_000_000},
	}

	fmt.Printf("  %-50s %12s %12s %10s\n", "Scenario", "Interface", "Concrete", "Savings")
	fmt.Printf("  %-50s %12s %12s %10s\n", "--------", "---------", "--------", "-------")

	for _, s := range scenarios {
		result := iface.CalculateCostProjection(s, costPerVCPUHour)
		fmt.Printf("  %-50s $%10.6f $%10.6f $%8.6f\n",
			result.Scenario.Name, result.InterfaceCost, result.ConcreteCost, result.Savings)
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
