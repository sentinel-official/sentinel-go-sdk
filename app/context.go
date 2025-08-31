package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
)

// SignalContext returns a context that is canceled when the process receives
// an interrupt (SIGINT) or termination (SIGTERM) signal.
func SignalContext(ctx context.Context) (context.Context, context.CancelCauseFunc) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		defer signal.Stop(signalChan)

		// Block until the first signal is received, then cancel the context
		// with a SignalError containing the received signal.
		sig := <-signalChan

		log.Debug("Received signal", "name", sig)
		cancel(&SignalError{Signal: sig})
	}()

	return ctx, cancel
}
