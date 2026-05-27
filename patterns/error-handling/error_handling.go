package error_handling

import (
	"errors"
	"fmt"
	"math/rand"
)

// ============================================================
// PATTERN 15: Error Handling Efficiency
// ============================================================
// Problem: errors.New() and fmt.Errorf() allocate on every call.
// On hot paths with high error rates, this creates millions of
// unnecessary heap allocations per day.
//
// This pattern demonstrates:
// 1. errors.New() allocates on every call
// 2. fmt.Errorf() allocates even more (format string + args)
// 3. Sentinel errors (var ErrX = errors.New(...)) — zero alloc on use
// 4. Custom error types with pre-allocated messages — zero alloc
// 5. Impact: 5% error rate on 100M req/day = 5M allocs/day eliminated
// ============================================================

// --- Error Creation Methods ---

// ErrorsNew creates a new error using errors.New() every time.
// Each call allocates a new error object on the heap.
func ErrorsNew() error {
	return errors.New("operation failed: resource not found")
}

// FmtErrorf creates a new error using fmt.Errorf() every time.
// Allocates even more: format string processing + argument boxing.
func FmtErrorf() error {
	return fmt.Errorf("operation failed: resource %s not found at %d", "user", 42)
}

// --- Sentinel Errors (Zero Allocation on Use) ---

// SentinelError is a package-level variable — allocated once at init,
// zero allocation on every subsequent use.
var SentinelError = errors.New("operation failed: resource not found")

// --- Pre-allocated Custom Error Type (Zero Allocation) ---

// CustomErrorType implements the error interface with a pre-built message.
// No allocation occurs when returning this error on hot paths.
type CustomErrorType struct {
	Code    int
	Message string
}

func (e *CustomErrorType) Error() string {
	return e.Message
}

// PreallocatedError is a package-level custom error — zero alloc on use.
var PreallocatedError = &CustomErrorType{
	Code:    404,
	Message: "operation failed: resource not found",
}

// --- Hot Path Simulation ---

// HotPathWithErrors simulates a function with a 5% error rate on a hot path.
// It demonstrates the allocation difference between error creation strategies.
func HotPathWithErrors(iterations int, errFunc func() error) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		// Simulate 5% error rate
		if rand.Intn(100) < 5 {
			globalErr = errFunc() // assign to global to prevent compiler optimization
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// HotPathWithSentinel simulates the same hot path but returns a sentinel error.
// Zero allocation regardless of error rate.
func HotPathWithSentinel(iterations int) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		if rand.Intn(100) < 5 {
			globalErr = SentinelError // No allocation — just a pointer copy
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// HotPathWithPreallocated simulates the hot path with a pre-allocated custom error.
// Zero allocation regardless of error rate.
func HotPathWithPreallocated(iterations int) (successes, failures int) {
	for i := 0; i < iterations; i++ {
		if rand.Intn(100) < 5 {
			globalErr = PreallocatedError // No allocation — just a pointer copy
			failures++
		} else {
			successes++
		}
	}
	return successes, failures
}

// globalErr prevents compiler from optimizing away error allocations.
var globalErr error
