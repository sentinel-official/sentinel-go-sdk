package utils

import (
	"context"
	"sync"
)

// AnyDoneContext returns a context that will be canceled when any of the provided
// contexts are canceled. It does not propagate values or deadlines from the inputs.
func AnyDoneContext(ctxs ...context.Context) (context.Context, context.CancelCauseFunc) {
	merged, cancel := context.WithCancelCause(context.Background())

	var once sync.Once
	for _, c := range ctxs {
		if c == nil {
			continue
		}

		go func(ctx context.Context) {
			select {
			case <-ctx.Done():
				once.Do(func() {
					cancel(ctx.Err())
				})
			case <-merged.Done():
			}
		}(c)
	}

	return merged, cancel
}
