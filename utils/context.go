package utils

import (
	"context"
)

// AnyDoneContext returns a context that will be canceled when any of the provided
// contexts are canceled. It does not propagate values or deadlines from the inputs.
func AnyDoneContext(ctxs ...context.Context) (context.Context, context.CancelFunc) {
	merged, cancel := context.WithCancel(context.Background())

	for _, c := range ctxs {
		if c == nil {
			continue
		}

		go func(ctx context.Context) {
			select {
			case <-ctx.Done():
				cancel()
			case <-merged.Done():
			}
		}(c)
	}

	return merged, cancel
}
