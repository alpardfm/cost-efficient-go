package main

import (
	"sort"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 17: Keyset pagination returns correct subset after cursor (elements > cursor, in order, limited to page size)
func TestProperty_KeysetPaginationCorrectSubset(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generate a fixed dataset for all tests
	users := generateUsers(1000)

	properties.Property("keyset pagination returns elements > cursor, in order, limited to page size", prop.ForAll(
		func(cursor int, pageSize int) bool {
			if pageSize < 1 {
				pageSize = 1
			}

			result := SimulateKeysetPagination(users, cursor, pageSize)

			// 1. All returned elements should have ID > cursor
			for _, u := range result {
				if u.ID <= cursor {
					return false
				}
			}

			// 2. Results should be in sorted order by ID
			if !sort.SliceIsSorted(result, func(i, j int) bool {
				return result[i].ID < result[j].ID
			}) {
				return false
			}

			// 3. Result size should be <= pageSize
			if len(result) > pageSize {
				return false
			}

			// 4. If there are elements after cursor, result should not be empty
			// (unless cursor is beyond all data)
			hasElementsAfterCursor := false
			for _, u := range users {
				if u.ID > cursor {
					hasElementsAfterCursor = true
					break
				}
			}
			if hasElementsAfterCursor && len(result) == 0 {
				return false
			}

			return true
		},
		gen.IntRange(0, 1050),
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}
