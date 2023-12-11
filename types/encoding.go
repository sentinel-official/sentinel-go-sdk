package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkstd "github.com/cosmos/cosmos-sdk/std"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	vpntypes "github.com/sentinel-official/hub/x/vpn/types"
)

// NewInterfaceRegistry initializes and returns a new InterfaceRegistry with registered interfaces
// for Cosmos SDK and Sentinel Hub modules.
// It covers interfaces for standard SDK types, authentication types, vesting types,
// authorization, fee grants, and VPN types used across the Cosmos SDK and Sentinel Hub ecosystem.
func NewInterfaceRegistry() codectypes.InterfaceRegistry {
	// Create a new InterfaceRegistry.
	registry := codectypes.NewInterfaceRegistry()

	// Register interfaces for Cosmos SDK modules.
	sdkstd.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	authvestingtypes.RegisterInterfaces(registry)
	authz.RegisterInterfaces(registry)
	feegrant.RegisterInterfaces(registry)

	// Register interfaces for Sentinel Hub modules.
	vpntypes.RegisterInterfaces(registry)

	// Return the populated InterfaceRegistry.
	return registry
}
