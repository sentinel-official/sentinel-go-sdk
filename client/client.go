package client

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

// BaseClient contains all necessary components for transaction handling, query management, and configuration settings.
type BaseClient struct {
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

// NewBaseClient initializes a new BaseClient instance.
func NewBaseClient() *BaseClient {
	// Create a codec for encoding/decoding protocol buffer messages.
	protoCodec := types.NewProtoCodec()
	txConfig := tx.NewTxConfig(protoCodec, tx.DefaultSignModes)

	// Initialize BaseClient with default values and configurations.
	bc := &BaseClient{}
	bc.WithProtoCodec(protoCodec)
	bc.WithTxConfig(txConfig)

	return bc
}

// TxFromName returns the transaction sender name set in the base client.
func (c *BaseClient) TxFromName() string {
	return c.txFromName
}

// WithKeyring assigns the keyring to the BaseClient and returns the updated BaseClient.
func (c *BaseClient) WithKeyring(keyring keyring.Keyring) *BaseClient {
	c.keyring = keyring
	return c
}

// WithProtoCodec sets the protobuf codec and returns the updated BaseClient.
func (c *BaseClient) WithProtoCodec(protoCodec codec.ProtoCodecMarshaler) *BaseClient {
	c.protoCodec = protoCodec
	return c
}

// WithQueryProve sets the prove flag for queries and returns the updated BaseClient.
func (c *BaseClient) WithQueryProve(prove bool) *BaseClient {
	c.queryProve = prove
	return c
}

// WithQueryRetries sets the number of retries for queries and returns the updated BaseClient.
func (c *BaseClient) WithQueryRetries(retries uint) *BaseClient {
	c.queryRetries = retries
	return c
}

// WithQueryRetryDelay sets the retry delay duration for queries and returns the updated BaseClient.
func (c *BaseClient) WithQueryRetryDelay(delay time.Duration) *BaseClient {
	c.queryRetryDelay = delay
	return c
}

// WithRPCAddr sets the RPC server address and returns the updated BaseClient.
func (c *BaseClient) WithRPCAddr(rpcAddr string) *BaseClient {
	c.rpcAddr = rpcAddr
	return c
}

// WithRPCChainID sets the blockchain chain ID and returns the updated BaseClient.
func (c *BaseClient) WithRPCChainID(chainID string) *BaseClient {
	c.rpcChainID = chainID
	return c
}

// WithRPCTimeout sets the RPC timeout duration and returns the updated BaseClient.
func (c *BaseClient) WithRPCTimeout(timeout time.Duration) *BaseClient {
	c.rpcTimeout = timeout
	return c
}

// WithTxConfig sets the transaction configuration and returns the updated BaseClient.
func (c *BaseClient) WithTxConfig(txConfig client.TxConfig) *BaseClient {
	c.txConfig = txConfig
	return c
}

// WithTxFeeGranterAddr sets the transaction fee granter address and returns the updated BaseClient.
func (c *BaseClient) WithTxFeeGranterAddr(addr cosmossdk.AccAddress) *BaseClient {
	c.txFeeGranterAddr = addr
	return c
}

// WithTxFees assigns transaction fees and returns the updated BaseClient.
func (c *BaseClient) WithTxFees(fees cosmossdk.Coins) *BaseClient {
	c.txFees = fees
	return c
}

// WithTxFromName sets the "from" name for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxFromName(name string) *BaseClient {
	c.txFromName = name
	return c
}

// WithTxGasAdjustment sets the gas adjustment factor for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxGasAdjustment(adjustment float64) *BaseClient {
	c.txGasAdjustment = adjustment
	return c
}

// WithTxGasPrices sets the gas prices for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxGasPrices(prices cosmossdk.DecCoins) *BaseClient {
	c.txGasPrices = prices
	return c
}

// WithTxGas sets the gas limit for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxGas(gas uint64) *BaseClient {
	c.txGas = gas
	return c
}

// WithTxMemo sets the memo for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxMemo(memo string) *BaseClient {
	c.txMemo = memo
	return c
}

// WithTxSimulateAndExecute sets the simulate and execute flag and returns the updated BaseClient.
func (c *BaseClient) WithTxSimulateAndExecute(simulate bool) *BaseClient {
	c.txSimulateAndExecute = simulate
	return c
}

// WithTxTimeoutHeight sets the timeout height for transactions and returns the updated BaseClient.
func (c *BaseClient) WithTxTimeoutHeight(height uint64) *BaseClient {
	c.txTimeoutHeight = height
	return c
}

// HTTP creates an HTTP client for the given RPC address and timeout configuration.
// Returns the HTTP client or an error if initialization fails.
func (c *BaseClient) HTTP() (*http.HTTP, error) {
	timeout := uint(c.rpcTimeout / time.Second)
	return http.NewWithTimeout(c.rpcAddr, "/websocket", timeout)
}

// NodeClient is a struct for interacting with nodes.
type NodeClient struct {
	*BaseClient
	addr     sentinelhub.NodeAddress
	insecure bool
	timeout  time.Duration
}

// NewNodeClient creates a new instance of NodeClient.
func NewNodeClient(base *BaseClient) *NodeClient {
	return &NodeClient{
		BaseClient: base,
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
