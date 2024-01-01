package options

import (
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/rpc/client/http"

	"github.com/sentinel-official/sentinel-go-sdk/v1/utils"
)

// Default values for query options.
const (
	DefaultQueryTimeout    = 15 * time.Second
	DefaultQueryMaxRetries = 60
	DefaultQueryWSEndpoint = "/websocket"
)

// QueryOptions defines a set of options for making queries, including RPC and WebSocket configurations.
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

// Query creates and returns a new QueryOptions instance with default values.
func Query() *QueryOptions {
	return &QueryOptions{
		MaxRetries: DefaultQueryMaxRetries,
		Timeout:    DefaultQueryTimeout,
		WSEndpoint: DefaultQueryWSEndpoint,
	}
}

// ABCIQueryOptions returns an ABCIQueryOptions instance based on the current QueryOptions.
func (q *QueryOptions) ABCIQueryOptions() client.ABCIQueryOptions {
	if q == nil {
		return client.ABCIQueryOptions{}
	}

	return client.ABCIQueryOptions{
		Height: q.Height,
		Prove:  q.Prove,
	}
}

// Client creates and returns an HTTP client based on the current QueryOptions.
func (q *QueryOptions) Client() (*http.HTTP, error) {
	if q == nil {
		return nil, errors.New("nil query options")
	}

	return http.NewWithTimeout(q.RPCAddr, q.WSEndpoint, utils.UIntSecondsFromDuration(q.Timeout))
}

// PageRequest returns a PageRequest instance based on the current QueryOptions.
func (q *QueryOptions) PageRequest() *query.PageRequest {
	if q == nil {
		return nil
	}

	return &query.PageRequest{
		Key:        q.PageKey,
		Offset:     q.PageOffset,
		Limit:      q.PageLimit,
		CountTotal: q.PageCountTotal,
		Reverse:    q.PageReverse,
	}
}

// WithHeight sets the height in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithHeight(v int64) *QueryOptions {
	q.Height = v
	return q
}

// WithMaxRetries sets the max retries in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithMaxRetries(v int) *QueryOptions {
	q.MaxRetries = v
	return q
}

// WithPageCountTotal sets the page count total flag in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithPageCountTotal(v bool) *QueryOptions {
	q.PageCountTotal = v
	return q
}

// WithPageKey sets the page key in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithPageKey(v []byte) *QueryOptions {
	q.PageKey = v
	return q
}

// WithPageLimit sets the page limit in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithPageLimit(v uint64) *QueryOptions {
	q.PageLimit = v
	return q
}

// WithPageOffset sets the page offset in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithPageOffset(v uint64) *QueryOptions {
	q.PageOffset = v
	return q
}

// WithPageReverse sets the page reverse flag in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithPageReverse(v bool) *QueryOptions {
	q.PageReverse = v
	return q
}

// WithProve sets the prove flag in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithProve(v bool) *QueryOptions {
	q.Prove = v
	return q
}

// WithRPCAddr sets the RPC address in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithRPCAddr(v string) *QueryOptions {
	q.RPCAddr = v
	return q
}

// WithTimeout sets the timeout in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithTimeout(v time.Duration) *QueryOptions {
	q.Timeout = v
	return q
}

// WithWSEndpoint sets the WebSocket endpoint in the current QueryOptions and returns the modified instance.
func (q *QueryOptions) WithWSEndpoint(v string) *QueryOptions {
	q.WSEndpoint = v
	return q
}
