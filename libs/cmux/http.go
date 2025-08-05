package cmux

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ListenAndServeTLS sets up a server that listens for both TLS and non-TLS traffic on the same address.
func ListenAndServeTLS(ctx context.Context, addr, certFile, keyFile string, handler http.Handler) error {
	// Load the TLS certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load tls certificate: %w", err)
	}

	// Create a TCP listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Create a cmux multiplexer
	mux := cmux.New(listener)

	// Define matchers for TLS and non-TLS traffic
	tlsMux := mux.Match(cmux.TLS())
	anyMux := mux.Match(cmux.Any())

	// Create an HTTP server specifically for TLS connections.
	// Suppress error logging to prevent spam from TLS handshake failures.
	tlsServer := &http.Server{
		Handler:  handler,
		ErrorLog: log.New(io.Discard, "", 0),
	}

	// Create a standard HTTP server for unencrypted traffic.
	anyServer := &http.Server{
		Handler: handler,
	}

	// Use errgroup to manage goroutines
	eg, ctx := errgroup.WithContext(ctx)

	// Serve TLS traffic
	eg.Go(func() error {
		// Reuse the TLS configuration
		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			Rand:         rand.Reader,
		}

		tlsMux := tls.NewListener(tlsMux, cfg)
		if err := tlsServer.Serve(tlsMux); err != nil {
			if utils.ErrorIs(err, cmux.ErrServerClosed) {
				return nil
			}

			return fmt.Errorf("failed to serve tls server: %w", err)
		}

		return nil
	})

	// Serve non-TLS traffic
	eg.Go(func() error {
		if err := anyServer.Serve(anyMux); err != nil {
			if utils.ErrorIs(err, cmux.ErrServerClosed) {
				return nil
			}

			return fmt.Errorf("failed to serve any server: %w", err)
		}

		return nil
	})

	// Start the multiplexer
	eg.Go(func() error {
		if err := mux.Serve(); err != nil {
			if utils.ErrorIs(err, net.ErrClosed) {
				return nil
			}

			return fmt.Errorf("failed to serve: %w", err)
		}

		return nil
	})

	// Handle shutdown on context cancellation
	eg.Go(func() error {
		<-ctx.Done()

		mux.Close()
		return nil
	})

	// Wait until all workers have completed
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
