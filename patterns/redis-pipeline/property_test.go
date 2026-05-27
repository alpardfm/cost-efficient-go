package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 16: Pipeline produces same results as individual operations (identical final state)
func TestProperty_PipelineSameResultsAsIndividual(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("pipeline produces identical state as individual ops", prop.ForAll(
		func(numOps int) bool {
			// Use zero latency for correctness testing
			mockIndividual := NewRedisMock(0*time.Millisecond, 10)
			mockPipeline := NewRedisMock(0*time.Millisecond, 10)

			// Individual SET operations
			for i := 0; i < numOps; i++ {
				key := fmt.Sprintf("key:%d", i)
				value := fmt.Sprintf("value:%d", i)
				mockIndividual.Set(key, value)
			}

			// Pipeline SET operations
			setCmds := make([]PipelineCommand, numOps)
			for i := 0; i < numOps; i++ {
				setCmds[i] = PipelineCommand{
					Op:    "SET",
					Key:   fmt.Sprintf("key:%d", i),
					Value: fmt.Sprintf("value:%d", i),
				}
			}
			mockPipeline.ExecPipeline(setCmds)

			// Verify both have identical state via GET
			for i := 0; i < numOps; i++ {
				key := fmt.Sprintf("key:%d", i)
				expectedValue := fmt.Sprintf("value:%d", i)

				indVal, indOk := mockIndividual.Get(key)
				pipVal, pipOk := mockPipeline.Get(key)

				if !indOk || !pipOk {
					return false
				}
				if indVal != expectedValue || pipVal != expectedValue {
					return false
				}
				if indVal != pipVal {
					return false
				}
			}
			return true
		},
		gen.IntRange(10, 100),
	))

	properties.TestingRun(t)
}
