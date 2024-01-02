package types

import (
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/proxy/vmess"
	"google.golang.org/protobuf/types/known/anypb"
)

// Proxy represents different types of proxies supported by the system.
type Proxy byte

const (
	// ProxyUnspecified represents an unspecified or unknown proxy type.
	ProxyUnspecified Proxy = 0x00 + iota
	// ProxyVMess represents the VMess proxy type.
	ProxyVMess
)

// String returns a human-readable string representation of the Proxy type.
func (p Proxy) String() string {
	switch p {
	case ProxyVMess:
		return "vmess"
	default:
		return ""
	}
}

// Tag returns a human-readable string representation of the Proxy type.
func (p Proxy) Tag() string {
	return p.String()
}

// Account generates an Any message containing the proxy account information.
func (p Proxy) Account(uid uuid.UUID) *anypb.Any {
	switch p {
	case ProxyVMess:
		return serial.ToTypedMessage(
			&vmess.Account{
				Id:      uid.String(),
				AlterId: 0,
				SecuritySettings: &protocol.SecurityConfig{
					Type: protocol.SecurityType_AUTO,
				},
				TestsEnabled: "",
			},
		)
	default:
		return nil
	}
}

// ProxyFromString converts a string representation to the corresponding Proxy type.
func ProxyFromString(s string) Proxy {
	switch s {
	case "vmess":
		return ProxyVMess
	default:
		return ProxyUnspecified
	}
}
