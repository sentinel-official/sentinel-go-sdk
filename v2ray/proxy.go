package v2ray

import (
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/proxy/vless"
	"github.com/v2fly/v2ray-core/v5/proxy/vmess"
	"google.golang.org/protobuf/types/known/anypb"
)

// ProxyProtocol is a custom type used to represent different proxy protocols.
type ProxyProtocol byte

// Constants for ProxyProtocol type with automatic incrementation for each protocol method.
const (
	ProxyProtocolUnspecified ProxyProtocol = iota // Default value for unspecified protocol
	ProxyProtocolVLess                            // ProxyProtocolVLess represents the VLess protocol
	ProxyProtocolVMess                            // ProxyProtocolVMess represents the VMess protocol
)

// String returns a string representation of the ProxyProtocol type.
func (p ProxyProtocol) String() string {
	switch p {
	case ProxyProtocolVLess:
		return "vless"
	case ProxyProtocolVMess:
		return "vmess"
	default:
		return "" // Return empty string for unspecified or unknown protocols
	}
}

// IsValid checks if the ProxyProtocol value is valid.
func (p ProxyProtocol) IsValid() bool {
	return p.String() != ""
}

// Account generates an account message based on the ProxyProtocol.
func (p ProxyProtocol) Account(uid uuid.UUID) *anypb.Any {
	switch p {
	case ProxyProtocolVLess:
		return serial.ToTypedMessage(
			&vless.Account{
				Id: uid.String(),
			},
		)
	case ProxyProtocolVMess:
		return serial.ToTypedMessage(
			&vmess.Account{
				Id: uid.String(),
			},
		)
	default:
		return nil
	}
}

// NewProxyProtocolFromString converts a string to a ProxyProtocol type.
func NewProxyProtocolFromString(v string) ProxyProtocol {
	switch v {
	case "vless":
		return ProxyProtocolVLess
	case "vmess":
		return ProxyProtocolVMess
	default:
		return ProxyProtocolUnspecified // Returns the default protocol if no match is found
	}
}
