package v2ray

import (
	"github.com/google/uuid"

	"github.com/sentinel-official/sentinel-go-sdk/libs/proxycmd"
	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// ProxyProtocol is a custom type used to represent different proxy protocols.
type ProxyProtocol byte

// Constants for ProxyProtocol type with automatic incrementation for each protocol method.
const (
	ProxyProtocolUnspecified ProxyProtocol = iota // Default value for unspecified protocol
	ProxyProtocolVLess                            // ProxyProtocolVLess represents the VLess protocol
	ProxyProtocolVMess                            // ProxyProtocolVMess represents the VMess protocol
)

// v2ray type URL header and per-protocol account type URLs.
const v2rayTypeURLHeader = "types.v2fly.org/"

const (
	typeVLESSAccount = v2rayTypeURLHeader + "v2ray.core.proxy.vless.Account"
	typeVMessAccount = v2rayTypeURLHeader + "v2ray.core.proxy.vmess.Account"
)

// dialect holds the v2ray-specific gRPC method paths and operation type URLs.
var dialect = proxycmd.Dialect{ //nolint:gochecknoglobals
	AlterInboundMethod:      "/v2ray.core.app.proxyman.command.HandlerService/AlterInbound",
	QueryStatsMethod:        "/v2ray.core.app.stats.command.StatsService/QueryStats",
	AddUserOperationType:    v2rayTypeURLHeader + "v2ray.core.app.proxyman.command.AddUserOperation",
	RemoveUserOperationType: v2rayTypeURLHeader + "v2ray.core.app.proxyman.command.RemoveUserOperation",
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

// String returns a string representation of the ProxyProtocol type.
func (p ProxyProtocol) String() string {
	switch p {
	case ProxyProtocolVLess:
		return "vless"
	case ProxyProtocolVMess:
		return "vmess"
	case ProxyProtocolUnspecified:
		return types.StringUnspecified
	default:
		return "" // Return empty string for unknown protocols
	}
}

// IsValid checks if the ProxyProtocol value is valid.
func (p ProxyProtocol) IsValid() bool {
	return p != ProxyProtocolUnspecified && p.String() != ""
}

// Account returns the type URL and encoded account bytes for the given UUID.
func (p ProxyProtocol) Account(uid uuid.UUID) (string, []byte) {
	switch p {
	case ProxyProtocolVLess:
		return typeVLESSAccount, proxycmd.VLESSAccount(uid.String(), "")
	case ProxyProtocolVMess:
		return typeVMessAccount, proxycmd.VMessAccount(uid.String())
	case ProxyProtocolUnspecified:
		return "", nil
	default:
		return "", nil
	}
}
