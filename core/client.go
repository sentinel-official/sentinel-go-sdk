package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/sentinel-official/sentinel-go-sdk/core/config"
	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// Client contains all necessary components for transaction handling, query management, and configuration settings.
type Client struct {
	keyring                  keyring.Keyring           // Keyring for managing private keys and signatures
	protoCodec               codec.ProtoCodecMarshaler // Used for marshaling and unmarshaling protobuf data
	queryHeight              int64                     // Query height for blockchain data
	queryProve               bool                      // Flag indicating whether to prove queries
	queryRetryAttempts       uint                      // Number of retry attempts for queries
	queryRetryDelay          time.Duration             // Delay between query retries
	rpcAddr                  string                    // RPC server address
	rpcChainID               string                    // The chain ID used to identify the blockchain network
	rpcHeaders               map[string]string         // Map to store custom RPC headers
	rpcTimeout               time.Duration             // RPC timeout duration
	txAuthzGranterAddr       cosmossdk.AccAddress      // Address that grants transaction authorization
	txBroadcastRetryAttempts uint                      // Number of retry attempts for transaction broadcast
	txBroadcastRetryDelay    time.Duration             // Delay between transaction broadcast retries
	txConfig                 client.TxConfig           // Configuration related to transactions (e.g., signing modes)
	txFeeGranterAddr         cosmossdk.AccAddress      // Address that grants transaction fees
	txFees                   cosmossdk.Coins           // Fees for transactions
	txFromName               string                    // Sender name for transactions
	txGasAdjustment          float64                   // Adjustment factor for gas estimation
	txGasPrices              cosmossdk.DecCoins        // Gas price settings for transactions
	txGas                    uint64                    // Gas limit for transactions
	txMemo                   string                    // Memo attached to transactions
	txQueryRetryAttempts     uint                      // Number of retry attempts for transaction queries
	txQueryRetryDelay        time.Duration             // Delay between transaction query retries
	txSimulateAndExecute     bool                      // Flag for simulating and executing transactions
	txTimeoutHeight          uint64                    // Transaction timeout height

	sealed bool

	fm sync.RWMutex
}

// NewClient initializes a new Client instance.
func NewClient() *Client {
	// Create a codec for encoding/decoding protocol buffer messages.
	protoCodec := types.NewProtoCodec()
	txConfig := tx.NewTxConfig(protoCodec, tx.DefaultSignModes)

	// Initialize Client with default values and configurations.
	c := &Client{}
	c.WithProtoCodec(protoCodec)
	c.WithTxConfig(txConfig)

	return c
}

// NewClientFromConfig creates a new Client instance based on the provided configuration.
func NewClientFromConfig(cfg *config.Config) (*Client, error) {
	c := NewClient().
		WithQueryProve(cfg.Query.GetProve()).
		WithQueryRetryAttempts(cfg.Query.GetRetryAttempts()).
		WithQueryRetryDelay(cfg.Query.GetRetryDelay()).
		WithRPCAddr(cfg.RPC.GetAddr()).
		WithRPCChainID(cfg.RPC.GetChainID()).
		WithRPCHeaders(cfg.RPC.GetHeaders()).
		WithRPCTimeout(cfg.RPC.GetTimeout()).
		WithTxAuthzGranterAddr(cfg.Tx.GetAuthzGranterAddr()).
		WithTxBroadcastRetryAttempts(cfg.Tx.GetBroadcastRetryAttempts()).
		WithTxBroadcastRetryDelay(cfg.Tx.GetBroadcastRetryDelay()).
		WithTxFeeGranterAddr(cfg.Tx.GetFeeGranterAddr()).
		WithTxFees(nil).
		WithTxFromName(cfg.Tx.GetFromName()).
		WithTxGasAdjustment(cfg.Tx.GetGasAdjustment()).
		WithTxGas(cfg.Tx.GetGas()).
		WithTxGasPrices(cfg.Tx.GetGasPrices()).
		WithTxMemo("").
		WithTxQueryRetryAttempts(cfg.Tx.GetQueryRetryAttempts()).
		WithTxQueryRetryDelay(cfg.Tx.GetQueryRetryDelay()).
		WithTxSimulateAndExecute(cfg.Tx.GetSimulateAndExecute()).
		WithTxTimeoutHeight(0)

	// Set up the keyring for the client
	if err := c.SetupKeyring(cfg.Keyring); err != nil {
		return nil, fmt.Errorf("setting up keyring: %w", err)
	}

	return c, nil
}

// Seal marks the client as sealed, preventing further modifications.
func (c *Client) Seal() *Client {
	c.sealed = true

	return c
}

// HTTP creates an HTTP client for the given RPC address and timeout configuration.
// Returns the HTTP client or an error if initialization fails.
func (c *Client) HTTP() (*http.HTTP, error) {
	c.fm.RLock()
	defer c.fm.RUnlock()

	// Create an HTTP client with the specified headers and timeout
	httpClient := newHTTPClient(c.rpcHeaders, c.rpcTimeout)

	// Create an HTTP-based RPC client
	v, err := http.NewWithClient(c.rpcAddr, "/websocket", httpClient)
	if err != nil {
		return nil, fmt.Errorf("creating RPC client: %w", err)
	}

	return v, nil
}

// MsgFromAddr returns the account address from which messages will be sent.
func (c *Client) MsgFromAddr() (cosmossdk.AccAddress, error) {
	c.fm.RLock()
	defer c.fm.RUnlock()

	if !c.txAuthzGranterAddr.Empty() {
		return c.txAuthzGranterAddr, nil
	}

	addr, err := c.KeyAddr(c.txFromName)
	if err != nil {
		return nil, fmt.Errorf("getting addr for key %q: %w", c.txFromName, err)
	}

	return addr, nil
}

// ProtoCodec returns the protobuf codec used for marshaling and unmarshaling data.
func (c *Client) ProtoCodec() codec.ProtoCodecMarshaler {
	c.fm.RLock()
	defer c.fm.RUnlock()

	return c.protoCodec
}

func (c *Client) SetRPCAddr(addr string) {
	c.fm.Lock()
	defer c.fm.Unlock()

	c.rpcAddr = addr
}

// WithKeyring assigns the keyring to the Client and returns the updated Client.
func (c *Client) WithKeyring(keyring keyring.Keyring) *Client {
	c.checkSealed()
	c.keyring = keyring

	return c
}

// WithProtoCodec sets the protobuf codec and returns the updated Client.
func (c *Client) WithProtoCodec(protoCodec codec.ProtoCodecMarshaler) *Client {
	c.checkSealed()
	c.protoCodec = protoCodec

	return c
}

// WithQueryProve sets the prove flag for queries and returns the updated Client.
func (c *Client) WithQueryProve(prove bool) *Client {
	c.checkSealed()
	c.queryProve = prove

	return c
}

// WithQueryRetryAttempts sets the number of retry attempts for queries and returns the updated Client.
func (c *Client) WithQueryRetryAttempts(attempts uint) *Client {
	c.checkSealed()
	c.queryRetryAttempts = attempts

	return c
}

// WithQueryRetryDelay sets the retry delay duration for queries and returns the updated Client.
func (c *Client) WithQueryRetryDelay(delay time.Duration) *Client {
	c.checkSealed()
	c.queryRetryDelay = delay

	return c
}

// WithRPCAddr sets the RPC server address and returns the updated Client.
func (c *Client) WithRPCAddr(rpcAddr string) *Client {
	c.checkSealed()
	c.rpcAddr = rpcAddr

	return c
}

// WithRPCChainID sets the blockchain chain ID and returns the updated Client.
func (c *Client) WithRPCChainID(chainID string) *Client {
	c.checkSealed()
	c.rpcChainID = chainID

	return c
}

// WithRPCHeaders sets the custom RPC headers and returns the updated Client.
func (c *Client) WithRPCHeaders(headers map[string]string) *Client {
	c.checkSealed()
	c.rpcHeaders = headers

	return c
}

// WithRPCTimeout sets the RPC timeout duration and returns the updated Client.
func (c *Client) WithRPCTimeout(timeout time.Duration) *Client {
	c.checkSealed()
	c.rpcTimeout = timeout

	return c
}

// WithTxAuthzGranterAddr sets the transaction authorization granter address and returns the updated Client.
func (c *Client) WithTxAuthzGranterAddr(addr cosmossdk.AccAddress) *Client {
	c.checkSealed()
	c.txAuthzGranterAddr = addr

	return c
}

// WithTxBroadcastRetryAttempts sets the number of retry attempts for broadcasting transactions and returns the updated Client.
func (c *Client) WithTxBroadcastRetryAttempts(attempts uint) *Client {
	c.checkSealed()
	c.txBroadcastRetryAttempts = attempts

	return c
}

// WithTxBroadcastRetryDelay sets the retry delay duration for broadcasting transactions and returns the updated Client.
func (c *Client) WithTxBroadcastRetryDelay(delay time.Duration) *Client {
	c.checkSealed()
	c.txBroadcastRetryDelay = delay

	return c
}

// WithTxConfig sets the transaction configuration and returns the updated Client.
func (c *Client) WithTxConfig(txConfig client.TxConfig) *Client {
	c.checkSealed()
	c.txConfig = txConfig

	return c
}

// WithTxFeeGranterAddr sets the transaction fee granter address and returns the updated Client.
func (c *Client) WithTxFeeGranterAddr(addr cosmossdk.AccAddress) *Client {
	c.checkSealed()
	c.txFeeGranterAddr = addr

	return c
}

// WithTxFees assigns transaction fees and returns the updated Client.
func (c *Client) WithTxFees(fees cosmossdk.Coins) *Client {
	c.checkSealed()
	c.txFees = fees

	return c
}

// WithTxFromName sets the "from" name for transactions and returns the updated Client.
func (c *Client) WithTxFromName(name string) *Client {
	c.checkSealed()
	c.txFromName = name

	return c
}

// WithTxGasAdjustment sets the gas adjustment factor for transactions and returns the updated Client.
func (c *Client) WithTxGasAdjustment(adjustment float64) *Client {
	c.checkSealed()
	c.txGasAdjustment = adjustment

	return c
}

// WithTxGasPrices sets the gas prices for transactions and returns the updated Client.
func (c *Client) WithTxGasPrices(prices cosmossdk.DecCoins) *Client {
	c.checkSealed()
	c.txGasPrices = prices

	return c
}

// WithTxGas sets the gas limit for transactions and returns the updated Client.
func (c *Client) WithTxGas(gas uint64) *Client {
	c.checkSealed()
	c.txGas = gas

	return c
}

// WithTxMemo sets the memo for transactions and returns the updated Client.
func (c *Client) WithTxMemo(memo string) *Client {
	c.checkSealed()
	c.txMemo = memo

	return c
}

// WithTxQueryRetryAttempts sets the number of retry attempts for transaction queries and returns the updated Client.
func (c *Client) WithTxQueryRetryAttempts(attempts uint) *Client {
	c.checkSealed()
	c.txQueryRetryAttempts = attempts

	return c
}

// WithTxQueryRetryDelay sets the retry delay duration for transaction queries and returns the updated Client.
func (c *Client) WithTxQueryRetryDelay(delay time.Duration) *Client {
	c.checkSealed()
	c.txQueryRetryDelay = delay

	return c
}

// WithTxSimulateAndExecute sets the simulate and execute flag and returns the updated Client.
func (c *Client) WithTxSimulateAndExecute(simulate bool) *Client {
	c.checkSealed()
	c.txSimulateAndExecute = simulate

	return c
}

// WithTxTimeoutHeight sets the timeout height for transactions and returns the updated Client.
func (c *Client) WithTxTimeoutHeight(height uint64) *Client {
	c.checkSealed()
	c.txTimeoutHeight = height

	return c
}

// checkSealed verifies if the client is sealed to prevent modification.
func (c *Client) checkSealed() {
	if c.sealed {
		panic(errors.New("client is sealed"))
	}
}
