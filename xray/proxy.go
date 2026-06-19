package xray

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"

	"github.com/sentinel-official/sentinel-go-sdk/libs/proxycmd"
	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// ShadowsocksMethod is the Shadowsocks 2022 method used for multi-user inbounds.
// Only blake3-aes-*-gcm methods support multiple users; aes-256-gcm needs 32-byte keys.
const ShadowsocksMethod = "2022-blake3-aes-256-gcm"

// shadowsocksKeyLength is the byte length of a Shadowsocks 2022 aes-256-gcm key.
const shadowsocksKeyLength = 32

// ProxyProtocol is a custom type used to represent different proxy protocols.
type ProxyProtocol byte

// Constants for ProxyProtocol type with automatic incrementation for each protocol method.
const (
	ProxyProtocolUnspecified     ProxyProtocol = iota // Default value for unspecified protocol
	ProxyProtocolVLess                                // ProxyProtocolVLess represents the VLess protocol
	ProxyProtocolVMess                                // ProxyProtocolVMess represents the VMess protocol
	ProxyProtocolTrojan                               // ProxyProtocolTrojan represents the Trojan protocol
	ProxyProtocolShadowsocks2022                      // ProxyProtocolShadowsocks2022 represents the Shadowsocks 2022 protocol
)

// Type URL constants for xray-core protobuf account messages.
const (
	typeVLESSAccount       = "xray.proxy.vless.Account"
	typeVMessAccount       = "xray.proxy.vmess.Account"
	typeTrojanAccount      = "xray.proxy.trojan.Account"
	typeShadowsocksAccount = "xray.proxy.shadowsocks_2022.Account"
)

// dialect holds the xray-core gRPC method paths and operation type URLs.
var dialect = proxycmd.Dialect{ //nolint:gochecknoglobals
	AlterInboundMethod:      "/xray.app.proxyman.command.HandlerService/AlterInbound",
	QueryStatsMethod:        "/xray.app.stats.command.StatsService/QueryStats",
	AddUserOperationType:    "xray.app.proxyman.command.AddUserOperation",
	RemoveUserOperationType: "xray.app.proxyman.command.RemoveUserOperation",
}

// NewProxyProtocolFromString converts a string to a ProxyProtocol type.
func NewProxyProtocolFromString(v string) ProxyProtocol {
	switch v {
	case "vless":
		return ProxyProtocolVLess
	case "vmess":
		return ProxyProtocolVMess
	case "trojan":
		return ProxyProtocolTrojan
	case "shadowsocks-2022":
		return ProxyProtocolShadowsocks2022
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
	case ProxyProtocolTrojan:
		return "trojan"
	case ProxyProtocolShadowsocks2022:
		return "shadowsocks-2022"
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

// Account returns the type URL and encoded account bytes for the given UUID and flow.
func (p ProxyProtocol) Account(uid uuid.UUID, flow Flow) (string, []byte) {
	switch p {
	case ProxyProtocolVLess:
		return typeVLESSAccount, proxycmd.VLESSAccount(uid.String(), flowString(flow))
	case ProxyProtocolVMess:
		return typeVMessAccount, proxycmd.VMessAccount(uid.String())
	case ProxyProtocolTrojan:
		return typeTrojanAccount, proxycmd.TrojanAccount(derivePassword(uid))
	case ProxyProtocolShadowsocks2022:
		return typeShadowsocksAccount, proxycmd.ShadowsocksAccount(deriveKey(uid))
	case ProxyProtocolUnspecified:
		return "", nil
	default:
		return "", nil
	}
}

// flowString returns the VLESS flow string sent to xray-core, empty unless Vision.
func flowString(flow Flow) string {
	if flow == FlowVision {
		return FlowVision.String()
	}

	return ""
}

// derivePassword deterministically derives a Trojan password from the UUID.
func derivePassword(uid uuid.UUID) string {
	return uid.String()
}

// deriveKey deterministically derives a Shadowsocks 2022 user PSK from the UUID.
func deriveKey(uid uuid.UUID) string {
	sum := sha256.Sum256(uid[:])

	return base64.StdEncoding.EncodeToString(sum[:])
}

// newKey generates a new random Shadowsocks 2022 server-level key.
func newKey() (string, error) {
	buf := make([]byte, shadowsocksKeyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// proxy holds the per-inbound parameters needed to add a user over gRPC.
type proxy struct {
	Protocol ProxyProtocol // The inbound proxy protocol.
	Flow     Flow          // The inbound VLESS flow control setting.
}
