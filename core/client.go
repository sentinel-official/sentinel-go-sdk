package core

import (
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	sentinelhub "github.com/sentinel-official/hub/v12/types"

	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// Client contains all necessary components for transaction handling, query management, and configuration settings.
type Client struct {
	keyring              keyring.Keyring           // Keyring for managing private keys and signatures
	protoCodec           codec.ProtoCodecMarshaler // Used for marshaling and unmarshaling protobuf data
	queryHeight          int64                     // Query height for blockchain data
	queryProve           bool                      // Flag indicating whether to prove queries
	queryRetries         uint                      // Number of retries for queries
	queryRetryDelay      time.Duration             // Delay between query retries
	rpcAddr              string                    // RPC server address
	rpcChainID           string                    // The chain ID used to identify the blockchain network
	rpcTimeout           time.Duration             // RPC timeout duration
	txConfig             client.TxConfig           // Configuration related to transactions (e.g., signing modes)
	txFeeGranterAddr     cosmossdk.AccAddress      // Address that grants transaction fees
	txFees               cosmossdk.Coins           // Fees for transactions
	txFromName           string                    // Sender name for transactions
	txGasAdjustment      float64                   // Adjustment factor for gas estimation
	txGasPrices          cosmossdk.DecCoins        // Gas price settings for transactions
	txGas                uint64                    // Gas limit for transactions
	txMemo               string                    // Memo attached to transactions
	txSimulateAndExecute bool                      // Flag for simulating and executing transactions
	txTimeoutHeight      uint64                    // Transaction timeout height
}

// NewBaseClient initializes a new Client instance.
func NewBaseClient() *Client {
	// Create a codec for encoding/decoding protocol buffer messages.
	protoCodec := types.NewProtoCodec()
	txConfig := tx.NewTxConfig(protoCodec, tx.DefaultSignModes)

	// Initialize Client with default values and configurations.
	bc := &Client{}
	bc.WithProtoCodec(protoCodec)
	bc.WithTxConfig(txConfig)

	return bc
}

// TxFromName returns the transaction sender name set in the base client.
func (c *Client) TxFromName() string {
	return c.txFromName
}

// WithKeyring assigns the keyring to the Client and returns the updated Client.
func (c *Client) WithKeyring(keyring keyring.Keyring) *Client {
	c.keyring = keyring
	return c
}

// WithProtoCodec sets the protobuf codec and returns the updated Client.
func (c *Client) WithProtoCodec(protoCodec codec.ProtoCodecMarshaler) *Client {
	c.protoCodec = protoCodec
	return c
}

// WithQueryProve sets the prove flag for queries and returns the updated Client.
func (c *Client) WithQueryProve(prove bool) *Client {
	c.queryProve = prove
	return c
}

// WithQueryRetries sets the number of retries for queries and returns the updated Client.
func (c *Client) WithQueryRetries(retries uint) *Client {
	c.queryRetries = retries
	return c
}

// WithQueryRetryDelay sets the retry delay duration for queries and returns the updated Client.
func (c *Client) WithQueryRetryDelay(delay time.Duration) *Client {
	c.queryRetryDelay = delay
	return c
}

// WithRPCAddr sets the RPC server address and returns the updated Client.
func (c *Client) WithRPCAddr(rpcAddr string) *Client {
	c.rpcAddr = rpcAddr
	return c
}

// WithRPCChainID sets the blockchain chain ID and returns the updated Client.
func (c *Client) WithRPCChainID(chainID string) *Client {
	c.rpcChainID = chainID
	return c
}

// WithRPCTimeout sets the RPC timeout duration and returns the updated Client.
func (c *Client) WithRPCTimeout(timeout time.Duration) *Client {
	c.rpcTimeout = timeout
	return c
}

// WithTxConfig sets the transaction configuration and returns the updated Client.
func (c *Client) WithTxConfig(txConfig client.TxConfig) *Client {
	c.txConfig = txConfig
	return c
}

// WithTxFeeGranterAddr sets the transaction fee granter address and returns the updated Client.
func (c *Client) WithTxFeeGranterAddr(addr cosmossdk.AccAddress) *Client {
	c.txFeeGranterAddr = addr
	return c
}

// WithTxFees assigns transaction fees and returns the updated Client.
func (c *Client) WithTxFees(fees cosmossdk.Coins) *Client {
	c.txFees = fees
	return c
}

// WithTxFromName sets the "from" name for transactions and returns the updated Client.
func (c *Client) WithTxFromName(name string) *Client {
	c.txFromName = name
	return c
}

// WithTxGasAdjustment sets the gas adjustment factor for transactions and returns the updated Client.
func (c *Client) WithTxGasAdjustment(adjustment float64) *Client {
	c.txGasAdjustment = adjustment
	return c
}

// WithTxGasPrices sets the gas prices for transactions and returns the updated Client.
func (c *Client) WithTxGasPrices(prices cosmossdk.DecCoins) *Client {
	c.txGasPrices = prices
	return c
}

// WithTxGas sets the gas limit for transactions and returns the updated Client.
func (c *Client) WithTxGas(gas uint64) *Client {
	c.txGas = gas
	return c
}

// WithTxMemo sets the memo for transactions and returns the updated Client.
func (c *Client) WithTxMemo(memo string) *Client {
	c.txMemo = memo
	return c
}

// WithTxSimulateAndExecute sets the simulate and execute flag and returns the updated Client.
func (c *Client) WithTxSimulateAndExecute(simulate bool) *Client {
	c.txSimulateAndExecute = simulate
	return c
}

// WithTxTimeoutHeight sets the timeout height for transactions and returns the updated Client.
func (c *Client) WithTxTimeoutHeight(height uint64) *Client {
	c.txTimeoutHeight = height
	return c
}

// HTTP creates an HTTP client for the given RPC address and timeout configuration.
// Returns the HTTP client or an error if initialization fails.
func (c *Client) HTTP() (*http.HTTP, error) {
	timeout := uint(c.rpcTimeout / time.Second)
	return http.NewWithTimeout(c.rpcAddr, "/websocket", timeout)
}

// NodeClient is a struct for interacting with nodes.
type NodeClient struct {
	*Client
	addr     sentinelhub.NodeAddress
	insecure bool
	timeout  time.Duration
}

// NewNodeClient creates a new instance of NodeClient.
func NewNodeClient(base *Client) *NodeClient {
	return &NodeClient{
		Client: base,
	}
}

// WithAddr sets the address of the NodeClient and returns the updated instance.
func (c *NodeClient) WithAddr(addr sentinelhub.NodeAddress) *NodeClient {
	c.addr = addr
	return c
}

// WithInsecure sets the insecure flag of the NodeClient and returns the updated instance.
func (c *NodeClient) WithInsecure(insecure bool) *NodeClient {
	c.insecure = insecure
	return c
}

// WithTimeout sets the timeout of the NodeClient and returns the updated instance.
func (c *NodeClient) WithTimeout(timeout time.Duration) *NodeClient {
	c.timeout = timeout
	return c
}
