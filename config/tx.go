package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/pflag"
)

// TxConfig defines the configuration for transactions.
type TxConfig struct {
	AuthzGranterAddr       string  `mapstructure:"authz_granter_addr"`       // AuthzGranterAddr is the address of the entity granting authorization.
	BroadcastRetryAttempts uint    `mapstructure:"broadcast_retry_attempts"` // Number of times to retry broadcasting a transaction.
	BroadcastRetryDelay    string  `mapstructure:"broadcast_retry_delay"`    // Delay between broadcast retries.
	FeeGranterAddr         string  `mapstructure:"fee_granter_addr"`         // FeeGranterAddr is the address of the entity granting fees.
	FromName               string  `mapstructure:"from_name"`                // FromName is the name of the sender's account.
	GasAdjustment          float64 `mapstructure:"gas_adjustment"`           // GasAdjustment is the adjustment factor for gas estimation.
	GasPrices              string  `mapstructure:"gas_prices"`               // GasPrices is the price of gas for the transaction.
	Gas                    uint64  `mapstructure:"gas"`                      // Gas is the gas limit for the transaction.
	QueryRetryAttempts     uint    `mapstructure:"query_retry_attempts"`     // Number of times to retry querying a transaction.
	QueryRetryDelay        string  `mapstructure:"query_retry_delay"`        // Delay between query retries.
	SimulateAndExecute     bool    `mapstructure:"simulate_and_execute"`     // SimulateAndExecute indicates whether to simulate the transaction before execution.
}

// GetAuthzGranterAddr returns the AuthzGranterAddr field as AccAddress.
func (c *TxConfig) GetAuthzGranterAddr() types.AccAddress {
	if c.AuthzGranterAddr == "" {
		return nil
	}

	addr, err := types.AccAddressFromBech32(c.AuthzGranterAddr)
	if err != nil {
		panic(err)
	}

	return addr
}

// GetBroadcastRetryAttempts returns the BroadcastRetryAttempts field.
func (c *TxConfig) GetBroadcastRetryAttempts() uint {
	return c.BroadcastRetryAttempts
}

// GetBroadcastRetryDelay returns the BroadcastRetryDelay field as time.Duration.
func (c *TxConfig) GetBroadcastRetryDelay() time.Duration {
	v, err := time.ParseDuration(c.BroadcastRetryDelay)
	if err != nil {
		panic(err)
	}

	return v
}

// GetFeeGranterAddr returns the FeeGranterAddr field.
func (c *TxConfig) GetFeeGranterAddr() types.AccAddress {
	if c.FeeGranterAddr == "" {
		return nil
	}

	addr, err := types.AccAddressFromBech32(c.FeeGranterAddr)
	if err != nil {
		panic(err)
	}

	return addr
}

// GetFromName returns the FromName field.
func (c *TxConfig) GetFromName() string {
	return c.FromName
}

// GetGas returns the Gas field.
func (c *TxConfig) GetGas() uint64 {
	return c.Gas
}

// GetGasAdjustment returns the GasAdjustment field.
func (c *TxConfig) GetGasAdjustment() float64 {
	return c.GasAdjustment
}

// GetGasPrices returns the GasPrices field as DecCoins.
func (c *TxConfig) GetGasPrices() types.DecCoins {
	coins, err := types.ParseDecCoins(c.GasPrices)
	if err != nil {
		panic(err)
	}

	return coins
}

// GetQueryRetryAttempts returns the QueryRetryAttempts field.
func (c *TxConfig) GetQueryRetryAttempts() uint {
	return c.QueryRetryAttempts
}

// GetQueryRetryDelay returns the QueryRetryDelay field as time.Duration.
func (c *TxConfig) GetQueryRetryDelay() time.Duration {
	v, err := time.ParseDuration(c.QueryRetryDelay)
	if err != nil {
		panic(err)
	}

	return v
}

// GetSimulateAndExecute returns the SimulateAndExecute field.
func (c *TxConfig) GetSimulateAndExecute() bool {
	return c.SimulateAndExecute
}

// Validate ensures the TxConfig has valid fields.
func (c *TxConfig) Validate() error {
	// Validate AuthzGranterAddr if it's not empty.
	if c.AuthzGranterAddr != "" {
		if _, err := types.AccAddressFromBech32(c.AuthzGranterAddr); err != nil {
			return fmt.Errorf("invalid authz_granter_addr: %w", err)
		}
	}

	// Ensure BroadcastRetryAttempts is non-zero.
	if c.BroadcastRetryAttempts == 0 {
		return errors.New("broadcast_retry_attempts cannot be zero")
	}

	// Validate FeeGranterAddr if it's not empty.
	if c.FeeGranterAddr != "" {
		if _, err := types.AccAddressFromBech32(c.FeeGranterAddr); err != nil {
			return fmt.Errorf("invalid fee_granter_addr: %w", err)
		}
	}

	// Ensure FromName is not empty.
	if c.FromName == "" {
		return errors.New("from_name cannot be empty")
	}

	// Ensure GasAdjustment is not negative.
	if c.GasAdjustment < 0 {
		return errors.New("gas_adjustment cannot be negative")
	}

	// Validate GasPrices if it's not empty.
	if c.GasPrices != "" {
		if _, err := types.ParseDecCoins(c.GasPrices); err != nil {
			return fmt.Errorf("invalid gas_prices: %w", err)
		}
	}

	// Ensure QueryRetryAttempts is non-zero.
	if c.QueryRetryAttempts == 0 {
		return errors.New("query_retry_attempts cannot be zero")
	}

	return nil
}

// SetForFlags adds tx configuration flags to the specified FlagSet.
func (c *TxConfig) SetForFlags(f *pflag.FlagSet) {
	f.StringVar(&c.AuthzGranterAddr, "tx.authz-granter-addr", c.AuthzGranterAddr, "address of the entity granting authorization")
	f.UintVar(&c.BroadcastRetryAttempts, "tx.broadcast-retry-attempts", c.BroadcastRetryAttempts, "number of times to retry broadcasting a transaction")
	f.StringVar(&c.BroadcastRetryDelay, "tx.broadcast-retry-delay", c.BroadcastRetryDelay, "delay between transaction broadcast retries")
	f.StringVar(&c.FeeGranterAddr, "tx.fee-granter-addr", c.FeeGranterAddr, "address of the entity granting fees")
	f.StringVar(&c.FromName, "tx.from-name", c.FromName, "name of the sender's account")
	f.Uint64Var(&c.Gas, "tx.gas", c.Gas, "gas limit for the transaction")
	f.Float64Var(&c.GasAdjustment, "tx.gas-adjustment", c.GasAdjustment, "adjustment factor for gas estimation")
	f.StringVar(&c.GasPrices, "tx.gas-prices", c.GasPrices, "price of gas for the transaction")
	f.BoolVar(&c.SimulateAndExecute, "tx.simulate-and-execute", c.SimulateAndExecute, "simulate the transaction before execution")
	f.UintVar(&c.QueryRetryAttempts, "tx.query-retry-attempts", c.QueryRetryAttempts, "number of times to retry querying a transaction")
	f.StringVar(&c.QueryRetryDelay, "tx.query-retry-delay", c.QueryRetryDelay, "delay between transaction query retries")
}

// DefaultTxConfig creates a TxConfig with default values.
func DefaultTxConfig() *TxConfig {
	return &TxConfig{
		AuthzGranterAddr:       "",
		BroadcastRetryAttempts: 1,
		BroadcastRetryDelay:    "5s",
		FeeGranterAddr:         "",
		FromName:               "main",
		Gas:                    200_000,
		GasAdjustment:          1.0 + 1.0/6,
		GasPrices:              "0.1udvpn",
		QueryRetryAttempts:     30,
		QueryRetryDelay:        "1s",
		SimulateAndExecute:     true,
	}
}
