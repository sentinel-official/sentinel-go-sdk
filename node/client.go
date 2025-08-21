package node

import (
	"fmt"
	"time"

	"github.com/sentinel-official/sentinelhub/v12/types"

	"github.com/sentinel-official/sentinel-go-sdk/config"
	"github.com/sentinel-official/sentinel-go-sdk/core"
)

// Client is a struct for interacting with nodes.
type Client struct {
	*core.Client
	addr     types.NodeAddress
	fromName string
	insecure bool
	timeout  time.Duration
}

// NewClient creates a new instance of Client.
func NewClient(c *core.Client) *Client {
	return &Client{
		Client: c,
	}
}

// WithAddr sets the address of the Client and returns the updated instance.
func (c *Client) WithAddr(addr types.NodeAddress) *Client {
	c.addr = addr
	return c
}

// WithFromName sets the fromName of the Client and returns the updated instance.
func (c *Client) WithFromName(fromName string) *Client {
	c.fromName = fromName
	return c
}

// WithInsecure sets the insecure flag of the Client and returns the updated instance.
func (c *Client) WithInsecure(insecure bool) *Client {
	c.insecure = insecure
	return c
}

// WithTimeout sets the timeout of the Client and returns the updated instance.
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.timeout = timeout
	return c
}

// NewClientFromConfig creates a new Client instance based on the provided configuration.
func NewClientFromConfig(cfg *config.Config) (*Client, error) {
	c, err := core.NewClientFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating core client: %w", err)
	}

	fromName := cfg.Tx.GetFromName()
	if addr := cfg.Tx.GetAuthzGranterAddr(); !addr.Empty() {
		key, err := c.KeyForAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("retrieving key for authz_granter_addr %q: %w", addr.String(), err)
		}
		if key == nil {
			return nil, fmt.Errorf("key for authz_granter_addr %q does not exist", addr.String())
		}

		fromName = key.Name
	}

	v := NewClient(c).
		WithAddr(nil).
		WithFromName(fromName).
		WithInsecure(false).
		WithTimeout(cfg.RPC.GetTimeout())

	return v, nil
}
