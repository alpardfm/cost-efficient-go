package main

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 5: All 4 concatenation methods produce identical output for any input []string
func TestProperty_StringConcatEquivalence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all 4 methods produce identical output", prop.ForAll(
		func(parts []string) bool {
			if len(parts) == 0 {
				// All methods should return empty string for empty input
				return ConcatPlus(parts) == "" &&
					ConcatSprintf(parts) == "" &&
					ConcatBuilder(parts) == "" &&
					ConcatBuffer(parts) == ""
			}

			expected := ConcatPlus(parts)
			return ConcatSprintf(parts) == expected &&
				ConcatBuilder(parts) == expected &&
				ConcatBuffer(parts) == expected
		},
		gen.SliceOf(gen.AnyString()).SuchThat(func(v interface{}) bool {
			s := v.([]string)
			return len(s) <= 100
		}),
	))

	properties.TestingRun(t)
}
