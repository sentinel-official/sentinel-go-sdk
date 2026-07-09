package proxycmd

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// Dialect describes the core-specific gRPC method paths and operation type URLs.
type Dialect struct {
	AlterInboundMethod      string // full gRPC method path for HandlerService/AlterInbound
	QueryStatsMethod        string // full gRPC method path for StatsService/QueryStats
	AddUserOperationType    string // type URL for the AddUserOperation wrapper
	RemoveUserOperationType string // type URL for the RemoveUserOperation wrapper
}

// Stat is a single named statistic returned by the stats service.
type Stat struct {
	Name  string
	Value int64
}

// AddUser adds a user to the given inbound tag over the HandlerService.
// account is the already-encoded protocol account wrapped as {type,value}; acctType is its type URL.
func AddUser(ctx context.Context, conn *grpc.ClientConn, d Dialect, tag, email, acctType string, acctValue []byte) error {
	// Encode account wrapper: TypedMessage{type_url=acctType, value=acctValue}.
	accountWrapper := typedMessage(acctType, acctValue)

	// Encode User{ email field2; account=accountWrapper field3 }.
	// level (field1) is omitted as it defaults to 0.
	var user []byte

	if email != "" {
		user = appendString(user, 2, email)
	}

	// Omit the account field when empty so an unset account is absent rather than present-but-empty.
	if len(accountWrapper) > 0 {
		user = appendMessage(user, 3, accountWrapper)
	}

	// Encode AddUserOperation{ user field1 }.
	var addOp []byte

	addOp = appendMessage(addOp, 1, user)

	// Encode operation wrapper: TypedMessage{type_url=AddUserOperationType, value=addOp}.
	opWrapper := typedMessage(d.AddUserOperationType, addOp)

	// Encode AlterInboundRequest{ tag field1; operation=opWrapper field2 }.
	var req []byte

	if tag != "" {
		req = appendString(req, 1, tag)
	}

	req = appendMessage(req, 2, opWrapper)

	var resp []byte
	if err := conn.Invoke(ctx, d.AlterInboundMethod, req, &resp, grpc.ForceCodec(rawCodec{})); err != nil {
		return fmt.Errorf("invoking %s: %w", d.AlterInboundMethod, err)
	}

	return nil
}

// RemoveUser removes the user with the given email from the inbound tag.
func RemoveUser(ctx context.Context, conn *grpc.ClientConn, d Dialect, tag, email string) error {
	// Encode RemoveUserOperation{ email field1 }.
	var rmOp []byte

	if email != "" {
		rmOp = appendString(rmOp, 1, email)
	}

	// Encode operation wrapper: TypedMessage{type_url=RemoveUserOperationType, value=rmOp}.
	opWrapper := typedMessage(d.RemoveUserOperationType, rmOp)

	// Encode AlterInboundRequest{ tag field1; operation=opWrapper field2 }.
	var req []byte

	if tag != "" {
		req = appendString(req, 1, tag)
	}

	req = appendMessage(req, 2, opWrapper)

	var resp []byte
	if err := conn.Invoke(ctx, d.AlterInboundMethod, req, &resp, grpc.ForceCodec(rawCodec{})); err != nil {
		return fmt.Errorf("invoking %s: %w", d.AlterInboundMethod, err)
	}

	return nil
}

// QueryStats queries the stats service for counters matching pattern.
func QueryStats(ctx context.Context, conn *grpc.ClientConn, d Dialect, pattern string, reset bool) ([]Stat, error) {
	// Encode QueryStatsRequest{ pattern field1; reset field2(bool) }.
	// Omit zero-value fields per proto3 default-omission.
	var req []byte

	if pattern != "" {
		req = appendString(req, 1, pattern)
	}

	if reset {
		req = appendBool(req, 2, reset)
	}

	var resp []byte
	if err := conn.Invoke(ctx, d.QueryStatsMethod, req, &resp, grpc.ForceCodec(rawCodec{})); err != nil {
		return nil, fmt.Errorf("invoking %s: %w", d.QueryStatsMethod, err)
	}

	stats, err := parseQueryStatsResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("parsing query stats response: %w", err)
	}

	return stats, nil
}
