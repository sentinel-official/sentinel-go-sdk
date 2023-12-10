package http

import (
	"time"

	"github.com/tendermint/tendermint/rpc/client/http"

	"github.com/sentinel-official/sentinel-sdk/v1/client/options"
	"github.com/sentinel-official/sentinel-sdk/v1/utils"
)

// NewWithTimeout creates a new HTTP client with the provided RPC address, WebSocket endpoint,
// and timeout. It converts the timeout to seconds using utils.UIntSecondsFromDuration.
func NewWithTimeout(rpcAddr, wsEndpoint string, timeout time.Duration) (*http.HTTP, error) {
	return http.NewWithTimeout(rpcAddr, wsEndpoint, utils.UIntSecondsFromDuration(timeout))
}

// NewFromQueryOptions creates a new HTTP client using the options specified in QueryOptions.
func NewFromQueryOptions(opts *options.QueryOptions) (*http.HTTP, error) {
	return NewWithTimeout(opts.RPCAddr, opts.WSEndpoint, opts.Timeout)
}

// NewFromTxOptions creates a new HTTP client using the options specified in TxOptions.
func NewFromTxOptions(opts *options.TxOptions) (*http.HTTP, error) {
	return NewWithTimeout(opts.RPCAddr, opts.WSEndpoint, opts.Timeout)
}
