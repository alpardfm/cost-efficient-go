package main

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 6: Interface and concrete type produce identical results for any float64 input
func TestProperty_InterfaceAndConcreteIdenticalResults(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("interface and concrete produce same result", prop.ForAll(
		func(value float64) bool {
			processor := NewConcreteProcessor(2.5, 1.0)
			var iface ProcessorInterface = processor

			// Process via concrete type
			concResult := processor.Process(value)
			// Process via interface
			ifaceResult := iface.Process(value)

			if concResult != ifaceResult {
				return false
			}

			// Also test Transform
			concTransform := processor.Transform(value)
			ifaceTransform := iface.Transform(value)

			return concTransform == ifaceTransform
		},
		gen.Float64Range(-1e10, 1e10),
	))

	properties.TestingRun(t)
}
