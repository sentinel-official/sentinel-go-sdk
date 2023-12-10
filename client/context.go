package client

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// Context represents a context related to the Cosmos SDK with a ProtoCodec for encoding and decoding.
type Context struct {
	*codec.ProtoCodec
}

// NewContext creates a new context with the provided InterfaceRegistry for encoding and decoding messages.
func NewContext(ir codectypes.InterfaceRegistry) *Context {
	return &Context{
		ProtoCodec: codec.NewProtoCodec(ir),
	}
}
