package options

import (
	"time"

	"github.com/tendermint/tendermint/rpc/client"
)

// Default values for query options
const (
	DefaultQueryTimeout    = 15 * time.Second
	DefaultQueryMaxRetries = 60
	DefaultQueryWSEndpoint = "/websocket"
)

// QueryOptions represents options for making queries.
type QueryOptions struct {
	Height     int64         `json:"height,omitempty"`      // Block height for the query
	MaxRetries int           `json:"max_retries,omitempty"` // Maximum number of retries for the query
	Prove      bool          `json:"prove,omitempty"`       // Whether to include proofs in the query response
	RPCAddr    string        `json:"rpc_addr,omitempty"`    // RPC address for the query
	Timeout    time.Duration `json:"timeout,omitempty"`     // Timeout for the query
	WSEndpoint string        `json:"ws_endpoint,omitempty"` // WebSocket endpoint for the query
}

// Query initializes and returns a new QueryOptions with default values.
func Query() *QueryOptions {
	return &QueryOptions{
		MaxRetries: DefaultQueryMaxRetries,
		Timeout:    DefaultQueryTimeout,
		WSEndpoint: DefaultQueryWSEndpoint,
	}
}

// WithMaxRetries sets the maximum number of retries for the query.
func (q *QueryOptions) WithMaxRetries(v int) *QueryOptions {
	q.MaxRetries = v
	return q
}

// WithRPCAddr sets the RPC address for the query.
func (q *QueryOptions) WithRPCAddr(v string) *QueryOptions {
	q.RPCAddr = v
	return q
}

// WithTimeout sets the timeout for the query.
func (q *QueryOptions) WithTimeout(v time.Duration) *QueryOptions {
	q.Timeout = v
	return q
}

// WithProve sets whether to include proofs in the query response.
func (q *QueryOptions) WithProve(v bool) *QueryOptions {
	q.Prove = v
	return q
}

// WithHeight sets the block height for the query.
func (q *QueryOptions) WithHeight(v int64) *QueryOptions {
	q.Height = v
	return q
}

// WithWSEndpoint sets the WebSocket endpoint for the query.
func (q *QueryOptions) WithWSEndpoint(v string) *QueryOptions {
	q.WSEndpoint = v
	return q
}

// ABCIQueryOptions returns an ABCIQueryOptions instance based on the current QueryOptions.
func (q *QueryOptions) ABCIQueryOptions() client.ABCIQueryOptions {
	return client.ABCIQueryOptions{
		Height: q.Height,
		Prove:  q.Prove,
	}
}
