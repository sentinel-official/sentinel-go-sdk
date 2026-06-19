package xray

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/proxy/shadowsocks_2022"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"

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
	return p.String() != ""
}

// Account generates an account message based on the ProxyProtocol.
func (p ProxyProtocol) Account(uid uuid.UUID, flow Flow) *serial.TypedMessage {
	switch p {
	case ProxyProtocolVLess:
		return serial.ToTypedMessage(
			&vless.Account{
				Id:   uid.String(),
				Flow: flowString(flow),
			},
		)
	case ProxyProtocolVMess:
		return serial.ToTypedMessage(
			&vmess.Account{
				Id: uid.String(),
			},
		)
	case ProxyProtocolTrojan:
		return serial.ToTypedMessage(
			&trojan.Account{
				Password: derivePassword(uid),
			},
		)
	case ProxyProtocolShadowsocks2022:
		return serial.ToTypedMessage(
			&shadowsocks_2022.Account{
				Key: deriveKey(uid),
			},
		)
	case ProxyProtocolUnspecified:
		return nil
	default:
		return nil
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
	sum := sha256.Sum256(uid.Bytes())

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
