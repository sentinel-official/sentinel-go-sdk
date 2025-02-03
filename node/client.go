package node

import (
	"time"

	"github.com/sentinel-official/hub/v12/types"

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
