package safe

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

type GRPCConn struct {
	c      *grpc.ClientConn
	closed atomic.Bool
	mu     sync.Mutex
	wg     sync.WaitGroup
}

// Dial connects to the given gRPC target if not already connected.
// If a connection already exists, Dial is a no-op.
func (c *GRPCConn) Dial(target string, opts ...grpc.DialOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return errors.New("gRPC connection already closed")
	}
	if c.c != nil {
		return nil
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return fmt.Errorf("creating gRPC client for %q: %w", target, err)
	}

	c.c = conn
	return nil
}

// Acquire marks the connection as in use and returns it along with a release function.
// The caller must invoke the release function exactly once when done.
func (c *GRPCConn) Acquire() (*grpc.ClientConn, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.c == nil || c.closed.Load() {
		return nil, func() {}
	}

	c.wg.Add(1)
	return c.c, func() { c.wg.Done() }
}

// Close waits for all active users to finish, then closes the connection.
// If the connection is already closed or was never opened, Close is a no-op.
func (c *GRPCConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	// Wait for all users to finish
	c.wg.Wait()

	if c.c == nil {
		return nil
	}

	if err := c.c.Close(); err != nil {
		return fmt.Errorf("closing gRPC connection: %w", err)
	}

	c.c = nil
	return nil
}
