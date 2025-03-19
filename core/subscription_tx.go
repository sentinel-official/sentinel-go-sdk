package core

import (
	"context"
	"fmt"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sentinel-official/hub/v12/types"
	"github.com/sentinel-official/hub/v12/x/subscription/types/v3"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// SubscriptionStartSession initiates a session for a subscription. On success, it returns the session ID.
func (c *Client) SubscriptionStartSession(ctx context.Context, id uint64, nodeAddr types.NodeAddress) (uint64, error) {
	// Retrieve the sender address.
	fromAddr, err := c.KeyAddr(c.txFromName)
	if err != nil {
		return 0, fmt.Errorf("failed to get from addr: %w", err)
	}

	// Construct the session start request message for a subscription session.
	msgs := []cosmossdk.Msg{
		v3.NewMsgStartSessionRequest(fromAddr, id, nodeAddr),
	}

	// Broadcast the transaction and wait for its inclusion in a block.
	_, res, err := c.BroadcastTxBlock(ctx, msgs...)
	if err != nil {
		return 0, fmt.Errorf("subscription start session tx failed: %w", err)
	}

	// Extract and return the session ID from the transaction events.
	id, err = utils.IDFromEvents(res.TxResult.GetEvents(), &v3.EventCreateSession{})
	if err != nil {
		return 0, fmt.Errorf("failed to get id from events: %w", err)
	}

	return id, nil
}
