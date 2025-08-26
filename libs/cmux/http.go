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
	"sync/atomic"
	"time"

	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Server is a multiprotocol HTTP server that supports both
// HTTP and HTTPS traffic on the same TCP port using cmux.
type Server struct {
	addr     string       // TCP address to listen on (e.g., ":8080")
	certFile string       // Path to TLS certificate file
	keyFile  string       // Path to TLS private key file
	handler  http.Handler // HTTP handler for processing requests

	anyServer *http.Server // HTTP server for non-TLS (plain HTTP) traffic
	tlsServer *http.Server // HTTP server for TLS (HTTPS) traffic

	eg      *errgroup.Group // Goroutine group to manage concurrent servers
	mux     cmux.CMux       // cmux multiplexer for matching connections
	running atomic.Bool     // Indicates if the server is currently running
}

// NewServer initializes a new Server instance with the given address,
// TLS certificate and key files, and HTTP handler.
func NewServer(addr, certFile, keyFile string, handler http.Handler) *Server {
	return &Server{
		addr:     addr,
		certFile: certFile,
		keyFile:  keyFile,
		handler:  handler,
		eg:       &errgroup.Group{},
	}
}

// Start launches the server and begins handling both HTTP and HTTPS traffic.
func (s *Server) Start() (err error) {
	// Prevent starting the server more than once
	if s.running.Swap(true) {
		return fmt.Errorf("server already running")
	}

	defer func() {
		// Revert running status in case of any error
		if err != nil {
			s.running.Store(false)
		}
	}()

	// Load the TLS certificate and key from disk
	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		return fmt.Errorf("loading TLS X509 certificate key pair from %q and %q: %w", s.certFile, s.keyFile, err)
	}

	// Create a TCP listener on the configured address
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("creating TCP listener on %q: %w", s.addr, err)
	}

	// Create a cmux multiplexer to distinguish between TLS and non-TLS traffic
	s.mux = cmux.New(listener)

	// Use TLS matcher to separate TLS traffic
	tlsMux := s.mux.Match(cmux.TLS())

	// Catch-all matcher for everything else (non-TLS traffic)
	anyMux := s.mux.Match(cmux.Any())

	// Configure the HTTPS server with no error logging to avoid handshake noise
	s.tlsServer = &http.Server{
		Handler:  s.handler,
		ErrorLog: log.New(io.Discard, "", 0),
	}

	// Configure the plain HTTP server
	s.anyServer = &http.Server{
		Handler: s.handler,
	}

	// Start serving TLS traffic in a separate goroutine
	s.eg.Go(func() error {
		cfg := &tls.Config{
			Certificates: []tls.Certificate{cert}, // Load TLS certificate
			Rand:         rand.Reader,             // Use secure random for crypto
		}

		// Wrap TLS matcher in a TLS listener
		l := tls.NewListener(tlsMux, cfg)

		// Start HTTPS server
		if err := s.tlsServer.Serve(l); err != nil {
			if !utils.ErrorIs(err, cmux.ErrServerClosed, http.ErrServerClosed) {
				return fmt.Errorf("serving TLS server: %w", err)
			}
		}

		return nil
	})

	// Start serving non-TLS HTTP traffic
	s.eg.Go(func() error {
		if err := s.anyServer.Serve(anyMux); err != nil {
			if !utils.ErrorIs(err, cmux.ErrServerClosed, http.ErrServerClosed) {
				return fmt.Errorf("serving any server: %w", err)
			}
		}

		return nil
	})

	// Start the cmux multiplexer, which delegates connections to the correct server
	s.eg.Go(func() error {
		if err := s.mux.Serve(); err != nil {
			if !utils.ErrorIs(err, net.ErrClosed) {
				return fmt.Errorf("serving multiplexer: %w", err)
			}
		}

		return nil
	})

	return nil
}

// Wait blocks until all server goroutines have exited or an error occurs.
func (s *Server) Wait() error {
	if err := s.eg.Wait(); err != nil {
		return err
	}

	return nil
}

// Stop gracefully shuts down both the TLS and non-TLS servers and stops the multiplexer.
func (s *Server) Stop() error {
	// Only allow shutdown if the server is running
	if !s.running.Swap(false) {
		return nil
	}

	// Use a timeout context to allow graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shut down both servers
	_ = s.tlsServer.Shutdown(ctx)
	_ = s.anyServer.Shutdown(ctx)

	// Close the multiplexer listener to stop accepting new connections
	s.mux.Close()

	return nil
}
