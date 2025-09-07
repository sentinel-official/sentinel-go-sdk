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
	"time"

	"github.com/soheilhy/cmux"

	"github.com/sentinel-official/sentinel-go-sdk/process"
)

// Server is a multiprotocol HTTP server that supports both
// HTTP and HTTPS traffic on the same TCP port using cmux.
type Server struct {
	*process.Manager // Embedded process manager for handling lifecycle

	addr     string       // TCP address to listen on (e.g., ":8080")
	certFile string       // Path to TLS certificate file
	handler  http.Handler // HTTP handler for processing requests
	keyFile  string       // Path to TLS private key file

	cMux      cmux.CMux    // CMux multiplexer for matching connections
	anyServer *http.Server // HTTP server for non-TLS (plain HTTP) traffic
	tlsServer *http.Server // HTTP server for TLS (HTTPS) traffic
}

// NewServer initializes a new Server instance with the given address,
// TLS certificate and key files, and HTTP handler.
func NewServer(name, addr, certFile, keyFile string, handler http.Handler) *Server {
	return &Server{
		Manager:  process.NewManager(name),
		addr:     addr,
		certFile: certFile,
		handler:  handler,
		keyFile:  keyFile,
	}
}

// Setup prepares the server for operation.
func (s *Server) Setup(ctx context.Context) error {
	return s.Manager.Setup(ctx, nil)
}

// Start launches the server and begins handling both HTTP and HTTPS traffic.
func (s *Server) Start(parent context.Context) (context.Context, error) {
	return s.Manager.Start(parent, func(ctx context.Context) error {
		// Load the TLS certificate and key from disk
		cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return fmt.Errorf("loading TLS X509 certificate key pair from %q and %q: %w", s.certFile, s.keyFile, err)
		}

		// Create a new ListenConfig
		lc := &net.ListenConfig{}

		// Create a TCP listener on the configured address
		listener, err := lc.Listen(ctx, "tcp", s.addr)
		if err != nil {
			return fmt.Errorf("creating listener on %q: %w", s.addr, err)
		}

		// Create a cmux multiplexer to distinguish between TLS and non-TLS traffic
		s.cMux = cmux.New(listener)

		// Use TLS matcher to separate TLS traffic
		tlsMux := s.cMux.Match(cmux.TLS())

		// Catch-all matcher for everything else (non-TLS traffic)
		anyMux := s.cMux.Match(cmux.Any())

		// Configure the HTTPS server with no error logging to avoid handshake noise
		s.tlsServer = &http.Server{
			Handler:  s.handler,
			ErrorLog: log.New(io.Discard, "", 0),
		}

		// Configure the plain HTTP server
		s.anyServer = &http.Server{
			Handler: s.handler,
		}

		// Start the cmux multiplexer, which delegates connections to the correct server
		s.Go(ctx, func() error {
			if err := s.cMux.Serve(); err != nil {
				return err
			}

			return nil
		})

		// Start serving TLS traffic in a separate goroutine
		s.Go(ctx, func() error {
			cfg := &tls.Config{
				Certificates: []tls.Certificate{cert}, // Load TLS certificate
				Rand:         rand.Reader,             // Use secure random for crypto
			}

			// Wrap TLS matcher in a TLS listener
			l := tls.NewListener(tlsMux, cfg)

			// Start HTTPS server
			if err := s.tlsServer.Serve(l); err != nil {
				return err
			}

			return nil
		})

		// Start serving non-TLS HTTP traffic
		s.Go(ctx, func() error {
			if err := s.anyServer.Serve(anyMux); err != nil {
				return err
			}

			return nil
		})

		// Close CMux on context cancellation
		s.Go(ctx, func() error {
			defer func() {
				_ = listener.Close()

				if s.cMux != nil {
					s.cMux.Close()
				}
			}()

			<-ctx.Done()

			return ctx.Err()
		})

		return nil
	})
}

// Wait blocks until all server goroutines have exited or an error occurs.
func (s *Server) Wait(ctx context.Context) error {
	return s.Manager.Wait(ctx, nil)
}

// Stop gracefully shuts down both the TLS and non-TLS servers and stops the multiplexer.
func (s *Server) Stop() error {
	return s.Manager.Stop(func() error {
		// Close cmux first (unblocks sub-listeners)
		if s.cMux != nil {
			s.cMux.Close()
		}

		// Gracefully stop TLS server
		if s.tlsServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := s.tlsServer.Shutdown(ctx); err != nil {
				_ = s.tlsServer.Close()
			}
		}

		// Gracefully stop Any server
		if s.anyServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := s.anyServer.Shutdown(ctx); err != nil {
				_ = s.anyServer.Close()
			}
		}

		return nil
	})
}

// Cleanup releases any remaining resources associated with the server.
func (s *Server) Cleanup() error {
	return s.Manager.Cleanup(func() error {
		s.cMux = nil
		s.anyServer = nil
		s.tlsServer = nil

		return nil
	})
}
