package interface_dispatch

import (
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

// --- Cost Projection Helpers ---

// CostProjectionScenario represents a scenario for cost projection analysis.
type CostProjectionScenario struct {
	Name       string
	OpsPerDay  int64
	CallsPerOp int
}

// CostProjectionResult holds the result of a cost projection calculation.
type CostProjectionResult struct {
	Scenario      CostProjectionScenario
	InterfaceCost float64
	ConcreteCost  float64
	Savings       float64
}

// CalculateCostProjection calculates cost projection for a given scenario.
func CalculateCostProjection(scenario CostProjectionScenario, costPerVCPUHour float64) CostProjectionResult {
	interfaceOverheadNs := 2.0 // Conservative estimate per call
	concreteNs := 0.4          // Inlined concrete call

	totalCalls := scenario.OpsPerDay * int64(scenario.CallsPerOp)

	interfaceTimeS := float64(totalCalls) * interfaceOverheadNs / 1e9
	concreteTimeS := float64(totalCalls) * concreteNs / 1e9

	interfaceCostDay := (interfaceTimeS / 3600) * costPerVCPUHour
	concreteCostDay := (concreteTimeS / 3600) * costPerVCPUHour
	savingsDay := interfaceCostDay - concreteCostDay

	return CostProjectionResult{
		Scenario:      scenario,
		InterfaceCost: interfaceCostDay,
		ConcreteCost:  concreteCostDay,
		Savings:       savingsDay,
	}
}

// MeasureOverhead measures the overhead of interface dispatch vs concrete calls.
func MeasureOverhead(processor *ConcreteProcessor, iterations int) (concreteDuration, interfaceDuration time.Duration) {
	// Concrete type hot loop
	start := time.Now()
	HotLoopConcrete(processor, iterations)
	concreteDuration = time.Since(start)

	// Interface hot loop
	var iface ProcessorInterface = processor
	start = time.Now()
	HotLoopInterface(iface, iterations)
	interfaceDuration = time.Since(start)

	return concreteDuration, interfaceDuration
}
