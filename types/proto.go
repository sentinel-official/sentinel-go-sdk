package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/sentinel-official/sentinelhub/v12/x/vpn/types/v1"
)

// NewInterfaceRegistry initializes and returns a new InterfaceRegistry with registered interfaces.
func NewInterfaceRegistry() codectypes.InterfaceRegistry {
	// Create a new InterfaceRegistry instance.
	registry := codectypes.NewInterfaceRegistry()

	// Register Cosmos SDK module interfaces.
	std.RegisterInterfaces(registry)
	auth.RegisterInterfaces(registry)
	vesting.RegisterInterfaces(registry)
	authz.RegisterInterfaces(registry)
	bank.RegisterInterfaces(registry)
	feegrant.RegisterInterfaces(registry)

	// Register Sentinel Hub module interfaces.
	v1.RegisterInterfaces(registry)

	// Return the populated InterfaceRegistry.
	return registry
}

// NewProtoCodec creates and returns a new ProtoCodecMarshaler with a populated InterfaceRegistry.
func NewProtoCodec() codec.ProtoCodecMarshaler {
	// Initialize the InterfaceRegistry.
	registry := NewInterfaceRegistry()

	// Create and return a new ProtoCodecMarshaler.
	return codec.NewProtoCodec(registry)
}
