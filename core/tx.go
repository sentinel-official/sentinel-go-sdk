package core

import (
	"context"
	"fmt"

	core "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	// gRPC methods for simulating the transaction
	methodSimulate = "/cosmos.tx.v1beta1.Service/Simulate"
)

// calculateFees computes transaction fees based on gas prices and gas limit.
func calculateFees(gasPrices cosmossdk.DecCoins, gasLimit uint64) cosmossdk.Coins {
	fees := make(cosmossdk.Coins, len(gasPrices))
	for i, price := range gasPrices {
		fee := price.Amount.MulInt64(int64(gasLimit))
		fees[i] = cosmossdk.NewCoin(price.Denom, fee.Ceil().RoundInt())
	}

	return fees
}

// Simulate simulates the execution of a transaction before broadcasting it.
// Takes transaction bytes as input and returns the simulation response or an error.
func (c *Client) Simulate(ctx context.Context, buf []byte) (*tx.SimulateResponse, error) {
	var (
		resp tx.SimulateResponse
		req  = &tx.SimulateRequest{TxBytes: buf}
	)

	// Perform a gRPC query to simulate the transaction.
	if err := c.QueryGRPC(ctx, methodSimulate, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to query simulate: %w", err)
	}

	return &resp, nil
}

// gasSimulateTx calculates the gas usage of a transaction.
// Returns the gas used and any error encountered.
func (c *Client) gasSimulateTx(ctx context.Context, txb client.TxBuilder) (uint64, error) {
	// Encode the transaction into bytes.
	buf, err := c.txConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return 0, fmt.Errorf("failed to encode tx: %w", err)
	}

	// Simulate the transaction execution.
	res, err := c.Simulate(ctx, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to simulate tx: %w", err)
	}

	// Calculate the adjusted gas usage.
	return uint64(c.txGasAdjustment * float64(res.GasInfo.GasUsed)), nil
}

// broadcastTxSync broadcasts a transaction synchronously.
// Returns the broadcast result or an error if the operation fails.
func (c *Client) broadcastTxSync(ctx context.Context, txb client.TxBuilder) (*core.ResultBroadcastTx, error) {
	// Encode the transaction into bytes.
	buf, err := c.txConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	// Get the HTTP client for broadcasting.
	http, err := c.HTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client: %w", err)
	}

	// Broadcast the transaction synchronously.
	res, err := http.BroadcastTxSync(ctx, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	return res, nil
}

// signTx signs a transaction using the provided key and account information.
// Returns an error if the signing process fails.
func (c *Client) signTx(txb client.TxBuilder, key *keyring.Record, account auth.AccountI) error {
	// Prepare the single signature data.
	singleSignatureData := txsigning.SingleSignatureData{
		SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}

	// Retrieve the public key from the key record.
	pubKey, err := key.GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to retrieve public key: %w", err)
	}

	// Create the signature information.
	signature := txsigning.SignatureV2{
		PubKey:   pubKey,
		Data:     &singleSignatureData,
		Sequence: account.GetSequence(),
	}

	// Set the initial signature in the transaction builder.
	if err := txb.SetSignatures(signature); err != nil {
		return fmt.Errorf("failed to set initial signatures: %w", err)
	}

	// Prepare the signer data for signing the transaction.
	signerData := authsigning.SignerData{
		ChainID:       c.rpcChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}

	// Get the bytes to be signed.
	buf, err := c.txConfig.SignModeHandler().GetSignBytes(singleSignatureData.SignMode, signerData, txb.GetTx())
	if err != nil {
		return fmt.Errorf("failed to get tx sign bytes: %w", err)
	}

	// Sign the transaction bytes.
	buf, _, err = c.Sign(c.txFromName, buf)
	if err != nil {
		return fmt.Errorf("failed to sign tx bytes`: %w", err)
	}

	// Update the signature data with the signed bytes.
	singleSignatureData.Signature = buf
	signature.Data = &singleSignatureData

	// Set the updated signature in the transaction builder.
	if err := txb.SetSignatures(signature); err != nil {
		return fmt.Errorf("failed to set updated signatures: %w", err)
	}

	return nil
}

// prepareTx prepares a transaction for broadcasting by setting fees, gas, and other parameters.
// Returns the transaction builder and any error encountered.
func (c *Client) prepareTx(ctx context.Context, key *keyring.Record, account auth.AccountI, msgs []cosmossdk.Msg) (client.TxBuilder, error) {
	// Create a new transaction builder.
	txb := c.txConfig.NewTxBuilder()
	if err := txb.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("failed to set messages: %w", err)
	}

	// Set transaction parameters.
	txb.SetFeeAmount(c.txFees)
	txb.SetFeeGranter(c.txFeeGranterAddr)
	txb.SetGasLimit(c.txGas)
	txb.SetMemo(c.txMemo)
	txb.SetTimeoutHeight(c.txTimeoutHeight)

	if !c.txGasPrices.IsZero() {
		fees := calculateFees(c.txGasPrices, c.txGas)
		txb.SetFeeAmount(fees)
	}

	// Retrieve the public key from the key record.
	pubKey, err := key.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve public key: %w", err)
	}

	// Set an initial signature in the transaction builder.
	signature := txsigning.SignatureV2{
		PubKey: pubKey,
		Data: &txsigning.SingleSignatureData{
			SignMode: txsigning.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: account.GetSequence(),
	}

	if err := txb.SetSignatures(signature); err != nil {
		return nil, fmt.Errorf("failed to set initial signatures: %w", err)
	}

	// Simulate the transaction to calculate gas usage if required.
	if c.txSimulateAndExecute {
		gasLimit, err := c.gasSimulateTx(ctx, txb)
		if err != nil {
			return nil, fmt.Errorf("failed to simulate tx for gas estimation: %w", err)
		}

		txb.SetGasLimit(gasLimit)
		if !c.txGasPrices.IsZero() {
			fees := calculateFees(c.txGasPrices, gasLimit)
			txb.SetFeeAmount(fees)
		}
	}

	return txb, nil
}

// BroadcastTx broadcasts a signed transaction and returns the broadcast result or an error.
func (c *Client) BroadcastTx(ctx context.Context, msgs []cosmossdk.Msg) (*core.ResultBroadcastTx, error) {
	// Retrieve the signing key.
	key, err := c.Key(c.txFromName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key: %w", err)
	}

	// Get the sender's address from the key.
	accAddr, err := key.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve addr: %w", err)
	}

	// Retrieve the sender's account information.
	account, err := c.Account(ctx, accAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}

	// Prepare the transaction for broadcasting.
	txb, err := c.prepareTx(ctx, key, account, msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx for broadcast: %w", err)
	}

	// Sign the transaction.
	if err := c.signTx(txb, key, account); err != nil {
		return nil, fmt.Errorf("failed to sign tx for broadcast: %w", err)
	}

	// Broadcast the signed transaction synchronously.
	res, err := c.broadcastTxSync(ctx, txb)
	if err != nil {
		return nil, fmt.Errorf("failed to sync broadcast tx: %w", err)
	}

	return res, nil
}

// Tx retrieves a transaction from the blockchain using its hash.
// Returns the transaction result or an error.
func (c *Client) Tx(ctx context.Context, hash []byte) (*core.ResultTx, error) {
	// Get the HTTP client for querying the blockchain.
	http, err := c.HTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client: %w", err)
	}

	// Perform the query using the transaction hash.
	res, err := http.Tx(ctx, hash, c.queryProve)
	if err != nil {
		return nil, fmt.Errorf("failed to query tx: %w", err)
	}

	return res, nil
}
