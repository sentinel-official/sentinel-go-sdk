package options

import (
	"time"
)

// Default values for transaction options
const (
	DefaultTxBroadcastMode      = "sync"
	DefaultTxGasAdjustment      = 1.0 + (1.0 / 6)
	DefaultTxMaxRetries         = 60
	DefaultTxSimulateAndExecute = true
	DefaultTxTimeout            = 15 * time.Second
	DefaultTxWSEndpoint         = "/websocket"
)

// TxOptions represents options for a transaction
type TxOptions struct {
	BroadcastMode      string        `json:"broadcast_mode,omitempty"`
	ChainID            string        `json:"chain_id,omitempty"`
	FeeGranterAddr     string        `json:"fee_granter_addr,omitempty"`
	Fees               string        `json:"fees,omitempty"`
	GasAdjustment      float64       `json:"gas_adjustment,omitempty"`
	Gas                int64         `json:"gas,omitempty"`
	GasPrices          string        `json:"gas_prices,omitempty"`
	MaxRetries         int           `json:"max_retries,omitempty"`
	RPCAddr            string        `json:"rpc_addr,omitempty"`
	SignMode           string        `json:"sign_mode,omitempty"`
	SimulateAndExecute bool          `json:"simulate_and_execute,omitempty"`
	TimeoutHeight      int64         `json:"timeout_height,omitempty"`
	Timeout            time.Duration `json:"timeout,omitempty"`
	WSEndpoint         string        `json:"ws_endpoint,omitempty"`
}

// Tx creates a new TxOptions with default values
func Tx() *TxOptions {
	return &TxOptions{
		BroadcastMode:      DefaultTxBroadcastMode,
		GasAdjustment:      DefaultTxGasAdjustment,
		MaxRetries:         DefaultTxMaxRetries,
		SimulateAndExecute: DefaultTxSimulateAndExecute,
		Timeout:            DefaultTxTimeout,
		WSEndpoint:         DefaultTxWSEndpoint,
	}
}

// WithBroadcastMode sets the broadcast mode for the transaction
func (t *TxOptions) WithBroadcastMode(v string) *TxOptions {
	t.BroadcastMode = v
	return t
}

// WithChainID sets the chain ID for the transaction
func (t *TxOptions) WithChainID(v string) *TxOptions {
	t.ChainID = v
	return t
}

// WithFeeGranterAddr sets the fee granter address for the transaction
func (t *TxOptions) WithFeeGranterAddr(v string) *TxOptions {
	t.FeeGranterAddr = v
	return t
}

// WithFees sets the fees for the transaction
func (t *TxOptions) WithFees(v string) *TxOptions {
	t.Fees = v
	return t
}

// WithGasAdjustment sets the gas adjustment for the transaction
func (t *TxOptions) WithGasAdjustment(v float64) *TxOptions {
	t.GasAdjustment = v
	return t
}

// WithGas sets the gas limit for the transaction
func (t *TxOptions) WithGas(v int64) *TxOptions {
	t.Gas = v
	return t
}

// WithGasPrices sets the gas prices for the transaction
func (t *TxOptions) WithGasPrices(v string) *TxOptions {
	t.GasPrices = v
	return t
}

// WithRPCAddr sets the RPC address for the transaction
func (t *TxOptions) WithRPCAddr(v string) *TxOptions {
	t.RPCAddr = v
	return t
}

// WithSignMode sets the sign mode for the transaction
func (t *TxOptions) WithSignMode(v string) *TxOptions {
	t.SignMode = v
	return t
}

// WithSimulateAndExecute sets the simulate and execute flag for the transaction
func (t *TxOptions) WithSimulateAndExecute(v bool) *TxOptions {
	t.SimulateAndExecute = v
	return t
}

// WithTimeoutHeight sets the timeout height for the transaction
func (t *TxOptions) WithTimeoutHeight(v int64) *TxOptions {
	t.TimeoutHeight = v
	return t
}

// WithTimeout sets the timeout duration for the transaction
func (t *TxOptions) WithTimeout(v time.Duration) *TxOptions {
	t.Timeout = v
	return t
}

// WithWSEndpoint sets the WebSocket endpoint for the transaction
func (t *TxOptions) WithWSEndpoint(v string) *TxOptions {
	t.WSEndpoint = v
	return t
}
