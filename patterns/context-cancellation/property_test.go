package context_cancellation

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: expand-cost-efficient-go, Property 9: Context cancellation propagates to all children (cancelled parent → all children Done)
func TestProperty_ContextCancellationPropagatesToChildren(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("cancelled parent context causes all children to be done", prop.ForAll(
		func(numChildren int) bool {
			parent, cancel := context.WithCancel(context.Background())

			// Create a tree of child contexts
			children := make([]context.Context, numChildren)
			childCancels := make([]context.CancelFunc, numChildren)
			for i := 0; i < numChildren; i++ {
				children[i], childCancels[i] = context.WithCancel(parent)
			}
			defer func() {
				for _, c := range childCancels {
					c()
				}
			}()

			// Cancel the parent
			cancel()

			// All children should be done
			for _, child := range children {
				select {
				case <-child.Done():
					// Good — child is cancelled
				case <-time.After(100 * time.Millisecond):
					return false // Child did not get cancelled in time
				}
			}
			return true
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t)
}

// Feature: expand-cost-efficient-go, Property 10: Cancelled operations complete faster than uncancelled (duration < full chain)
func TestProperty_CancelledOperationsCompleteFaster(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("cancelled operation completes faster than full chain", prop.ForAll(
		func(cancelAfterMs int) bool {
			cancelAfter := time.Duration(cancelAfterMs) * time.Millisecond

			// Run with cancellation
			ctx, cancel := context.WithTimeout(context.Background(), cancelAfter)
			start := time.Now()
			result := CascadingCall(ctx)
			cancelledDuration := time.Since(start)
			cancel()

			// Run without cancellation (full chain)
			start = time.Now()
			fullResult := CascadingCall(context.Background())
			fullDuration := time.Since(start)

			// If the operation was actually cancelled (not completed normally),
			// it should have taken less time than the full chain
			if !result.Completed && fullResult.Completed {
				return cancelledDuration < fullDuration
			}

			// If cancel timeout is longer than full chain, both complete normally
			return true
		},
		gen.IntRange(10, 100),
	))

	properties.TestingRun(t)
}
