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
	Height         int64         `json:"height,omitempty"`
	MaxRetries     int           `json:"max_retries,omitempty"`
	PageCountTotal bool          `json:"page_count_total,omitempty"`
	PageKey        []byte        `json:"page_key,omitempty"`
	PageLimit      uint64        `json:"page_limit,omitempty"`
	PageOffset     uint64        `json:"page_offset,omitempty"`
	PageReverse    bool          `json:"page_reverse,omitempty"`
	Prove          bool          `json:"prove,omitempty"`
	RPCAddr        string        `json:"rpc_addr,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
	WSEndpoint     string        `json:"ws_endpoint,omitempty"`
}

// Query initializes and returns a new QueryOptions with default values.
func Query() *QueryOptions {
	return &QueryOptions{
		MaxRetries: DefaultQueryMaxRetries,
		Timeout:    DefaultQueryTimeout,
		WSEndpoint: DefaultQueryWSEndpoint,
	}
}

// ABCIQueryOptions returns an ABCIQueryOptions instance based on the current QueryOptions.
func (q *QueryOptions) ABCIQueryOptions() client.ABCIQueryOptions {
	return client.ABCIQueryOptions{
		Height: q.Height,
		Prove:  q.Prove,
	}
}

// WithHeight sets the block height for the query.
func (q *QueryOptions) WithHeight(v int64) *QueryOptions {
	q.Height = v
	return q
}

// WithMaxRetries sets the maximum number of retries for the query.
func (q *QueryOptions) WithMaxRetries(v int) *QueryOptions {
	q.MaxRetries = v
	return q
}

// WithPageCountTotal sets whether to include the total number of pages in the query response.
func (q *QueryOptions) WithPageCountTotal(v bool) *QueryOptions {
	q.PageCountTotal = v
	return q
}

// WithPageKey sets the page key for the query.
func (q *QueryOptions) WithPageKey(v []byte) *QueryOptions {
	q.PageKey = v
	return q
}

// WithPageLimit sets the page limit for the query.
func (q *QueryOptions) WithPageLimit(v uint64) *QueryOptions {
	q.PageLimit = v
	return q
}

// WithPageOffset sets the page offset for the query.
func (q *QueryOptions) WithPageOffset(v uint64) *QueryOptions {
	q.PageOffset = v
	return q
}

// WithPageReverse sets whether to reverse the order of the pages in the query response.
func (q *QueryOptions) WithPageReverse(v bool) *QueryOptions {
	q.PageReverse = v
	return q
}

// WithProve sets whether to include proofs in the query response.
func (q *QueryOptions) WithProve(v bool) *QueryOptions {
	q.Prove = v
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

// WithWSEndpoint sets the WebSocket endpoint for the query.
func (q *QueryOptions) WithWSEndpoint(v string) *QueryOptions {
	q.WSEndpoint = v
	return q
}
