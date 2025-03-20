package core

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/bytes"
	core "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// MsgFromAddr returns the account address from which messages will be sent.
func (c *Client) MsgFromAddr() (cosmossdk.AccAddress, error) {
	if !c.txAuthzGranterAddr.Empty() {
		return c.txAuthzGranterAddr, nil
	}

	addr, err := c.KeyAddr(c.txFromName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key addr for tx_from_name:%w", err)
	}

	return addr, nil
}

// calculateFees computes transaction fees based on the provided gas prices and gas limit.
func calculateFees(gasPrices cosmossdk.DecCoins, gasLimit uint64) cosmossdk.Coins {
	fees := make(cosmossdk.Coins, len(gasPrices))
	for i, price := range gasPrices {
		fee := price.Amount.MulInt64(int64(gasLimit))
		fees[i] = cosmossdk.NewCoin(price.Denom, fee.Ceil().RoundInt())
	}

	return fees
}

// gasSimulateTx simulates the execution of a transaction to estimate the gas usage.
func (c *Client) gasSimulateTx(ctx context.Context, txb client.TxBuilder) (uint64, error) {
	// Encode the transaction into bytes.
	buf, err := c.txConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return 0, fmt.Errorf("failed to encode tx: %w", err)
	}

	// Simulate the transaction execution to estimate gas usage.
	res, err := c.Simulate(ctx, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to simulate tx: %w", err)
	}

	// Apply the gas adjustment factor to the simulated gas used.
	return uint64(c.txGasAdjustment * float64(res.GasInfo.GasUsed)), nil
}

// prepareTx prepares a transaction for broadcasting by setting messages, fees, gas limit, memo, and other parameters.
func (c *Client) prepareTx(ctx context.Context, key *keyring.Record, acc auth.AccountI, msgs ...cosmossdk.Msg) (client.TxBuilder, error) {
	// Create a new transaction builder.
	txb := c.txConfig.NewTxBuilder()

	// Set the transaction messages.
	if err := txb.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("failed to set messages: %w", err)
	}

	// Set static transaction parameters.
	txb.SetFeeAmount(c.txFees)
	txb.SetFeeGranter(c.txFeeGranterAddr)
	txb.SetGasLimit(c.txGas)
	txb.SetMemo(c.txMemo)
	txb.SetTimeoutHeight(c.txTimeoutHeight)

	// If gas prices are provided (non-zero), recalculate fees based on the gas limit.
	if !c.txGasPrices.IsZero() {
		fees := calculateFees(c.txGasPrices, c.txGas)
		txb.SetFeeAmount(fees)
	}

	// Prepare the initial signature data with a nil signature.
	singleSignatureData := txsigning.SingleSignatureData{
		SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}

	// Retrieve the public key from the key record.
	pubKey, err := key.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from key: %w", err)
	}

	// Create the signature information with the account sequence.
	signature := txsigning.SignatureV2{
		PubKey:   pubKey,
		Data:     &singleSignatureData,
		Sequence: acc.GetSequence(),
	}

	// Set the initial (placeholder) signature in the transaction builder.
	if err := txb.SetSignatures(signature); err != nil {
		return nil, fmt.Errorf("failed to set initial signatures: %w", err)
	}

	// If simulation is enabled, simulate the transaction to recalculate the gas limit and fees.
	if c.txSimulateAndExecute {
		gasLimit, err := c.gasSimulateTx(ctx, txb)
		if err != nil {
			return nil, fmt.Errorf("failed to simulate tx for gas estimation: %w", err)
		}

		// Update the gas limit based on simulation.
		txb.SetGasLimit(gasLimit)

		// Recalculate fees if gas prices are provided.
		if !c.txGasPrices.IsZero() {
			fees := calculateFees(c.txGasPrices, gasLimit)
			txb.SetFeeAmount(fees)
		}
	}

	return txb, nil
}

// signTx signs a transaction using the provided key and account information.
func (c *Client) signTx(txb client.TxBuilder, key *keyring.Record, acc auth.AccountI) error {
	// Prepare the initial signature data with a nil signature.
	singleSignatureData := txsigning.SingleSignatureData{
		SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}

	// Retrieve the public key from the key record.
	pubKey, err := key.GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to get public key from key: %w", err)
	}

	// Create the signature information including the account sequence.
	signature := txsigning.SignatureV2{
		PubKey:   pubKey,
		Data:     &singleSignatureData,
		Sequence: acc.GetSequence(),
	}

	// Set the initial (placeholder) signature in the transaction builder.
	if err := txb.SetSignatures(signature); err != nil {
		return fmt.Errorf("failed to set initial signatures: %w", err)
	}

	// Prepare the signer data required for signing the transaction.
	signerData := authsigning.SignerData{
		ChainID:       c.rpcChainID,
		AccountNumber: acc.GetAccountNumber(),
		Sequence:      acc.GetSequence(),
	}

	// Obtain the bytes to be signed from the transaction builder.
	buf, err := c.txConfig.SignModeHandler().GetSignBytes(singleSignatureData.SignMode, signerData, txb.GetTx())
	if err != nil {
		return fmt.Errorf("failed to get tx sign bytes: %w", err)
	}

	// Sign the transaction bytes using the provided key (identified by c.txFromName).
	buf, _, err = c.Sign(c.txFromName, buf)
	if err != nil {
		return fmt.Errorf("failed to sign tx bytes: %w", err)
	}

	// Update the signature data with the generated signature.
	singleSignatureData.Signature = buf
	signature.Data = &singleSignatureData

	// Update the transaction builder with the final signature.
	if err := txb.SetSignatures(signature); err != nil {
		return fmt.Errorf("failed to set updated signatures: %w", err)
	}

	return nil
}

// broadcastTxSync broadcasts a signed transaction synchronously and returns the broadcast result.
func (c *Client) broadcastTxSync(ctx context.Context, msgs ...cosmossdk.Msg) (*core.ResultBroadcastTx, error) {
	// Retrieve the signing key using the configured sender name.
	key, err := c.Key(c.txFromName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	if key == nil {
		return nil, newErrNotFound(fmt.Errorf("key %s does not exist", c.txFromName))
	}

	// Get the sender's address from the key record.
	addr, err := key.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get addr from key: %w", err)
	}

	if !c.txAuthzGranterAddr.Empty() {
		execMsg := authz.NewMsgExec(addr, msgs)
		msgs = []cosmossdk.Msg{&execMsg}
	}

	// Validate each message and return an error if any fail.
	for i, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			return nil, fmt.Errorf("failed to validate message at index %d: %w", i, err)
		}
	}

	// Retrieve the sender's account information from the blockchain.
	acc, err := c.Account(ctx, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}
	if acc == nil {
		return nil, newErrNotFound(fmt.Errorf("acconut %s does not exist", addr))
	}

	// Prepare the transaction (set messages, fees, gas, etc.) for broadcasting.
	txb, err := c.prepareTx(ctx, key, acc, msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx: %w", err)
	}

	// Sign the transaction.
	if err := c.signTx(txb, key, acc); err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	// Encode the signed transaction into bytes.
	buf, err := c.txConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	// Get the HTTP client for broadcasting the transaction.
	http, err := c.HTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc client: %w", err)
	}

	// Broadcast the transaction synchronously via the HTTP client.
	res, err := http.BroadcastTxSync(ctx, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to sync broadcast tx: %w", err)
	}

	return res, nil
}

// BroadcastTxSync attempts to broadcast a transaction synchronously with retry logic.
func (c *Client) BroadcastTxSync(ctx context.Context, msgs ...cosmossdk.Msg) (*core.ResultBroadcastTx, error) {
	var err error
	var resp *core.ResultBroadcastTx

	// Define a function to perform the transaction broadcast.
	retryFunc := func() error {
		// Attempt to broadcast the transaction.
		resp, err = c.broadcastTxSync(ctx, msgs...)
		if err != nil {
			// Return nil if the error is related to a mempool cache issue.
			if IsTxInMempoolCacheError(err) {
				return nil
			}

			return err
		}

		return nil
	}

	// retryIfFunc determines whether a retry should occur based on the error.
	retryIfFunc := func(err error) bool {
		// Retry if the error is an account sequence mismatch.
		if IsWrongSequenceError(err) {
			return true
		}

		return false
	}

	// Retry broadcasting the transaction with defined attempts and delay.
	if err := retry.Do(
		retryFunc,
		retry.Attempts(c.txBroadcastRetryAttempts),
		retry.Delay(c.txBroadcastRetryDelay),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.RetryIf(retryIfFunc),
	); err != nil {
		return nil, fmt.Errorf("tx sync broadcast failed after retries: %w", err)
	}

	return resp, nil
}

// tx retrieves a transaction from the blockchain using its hash.
func (c *Client) tx(ctx context.Context, hash bytes.HexBytes) (*core.ResultTx, error) {
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

// Tx retrieves a transaction from the blockchain using its hash, with retry logic.
func (c *Client) Tx(ctx context.Context, hash bytes.HexBytes) (*core.ResultTx, error) {
	var err error
	var result *core.ResultTx

	// Define a function to perform the transaction query.
	retryFunc := func() error {
		result, err = c.tx(ctx, hash)
		if err != nil {
			return err
		}

		return nil
	}

	// retryIfFunc signals that a retry should occur on any error.
	retryIfFunc := func(err error) bool {
		return true
	}

	// Retry fetching the transaction.
	if err := retry.Do(
		retryFunc,
		retry.Attempts(c.txQueryRetryAttempts),
		retry.Delay(c.txQueryRetryDelay),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.RetryIf(retryIfFunc),
	); err != nil {
		return nil, fmt.Errorf("tx query failed after retries: %w", err)
	}

	return result, nil
}

// BroadcastTxBlock broadcasts a transaction and waits for it to be included in a block.
// It first calls BroadcastTxSync to send the transaction and then queries for the transaction result.
// Returns both the broadcast response and the transaction result or an error if any step fails.
func (c *Client) BroadcastTxBlock(ctx context.Context, msgs ...cosmossdk.Msg) (*core.ResultBroadcastTx, *core.ResultTx, error) {
	// Broadcast the transaction synchronously.
	resp, err := c.BroadcastTxSync(ctx, msgs...)
	if err != nil {
		return nil, nil, err
	}

	//  Ensure the transaction was accepted by the mempool.
	if resp.Code != abci.CodeTypeOK {
		err := fmt.Errorf("code=%d, codespace=%s, log=%s", resp.Code, resp.Codespace, resp.Log)
		return resp, nil, fmt.Errorf("tx sync broadcast failed: %w", err)
	}

	// Wait for the transaction to be included in a block.
	res, err := c.Tx(ctx, resp.Hash)
	if err != nil {
		return resp, nil, err
	}

	//  Ensure the transaction executed successfully.
	if !res.TxResult.IsOK() {
		err := fmt.Errorf("code=%d, codespace=%s, log=%s", res.TxResult.Code, res.TxResult.Codespace, res.TxResult.Log)
		return resp, res, fmt.Errorf("tx failed: %w", err)
	}

	// Return the broadcast response and transaction result.
	return resp, res, nil
}
