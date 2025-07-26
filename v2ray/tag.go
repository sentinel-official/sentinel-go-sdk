package v2ray

import (
	"fmt"

	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/proxy/vless"
	"github.com/v2fly/v2ray-core/v5/proxy/vmess"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
)

// Tag represents a composite data structure combining ProxyProtocol, TransportProtocol, and TransportSecurity.
type Tag struct {
	Port      *netip.Port       `json:"port"`
	Proxy     ProxyProtocol     `json:"proxy"`
	Security  TransportSecurity `json:"security"`
	Transport TransportProtocol `json:"transport"`
}

// String returns a string representation of the Tag.
func (t *Tag) String() string {
	return fmt.Sprintf("%s_%s_%s_%s", t.Port, t.Proxy, t.Security, t.Transport)
}

// Account generates an account message based on the ProxyProtocol stored in the Tag.
func (t *Tag) Account(uid uuid.UUID) *anypb.Any {
	switch t.Proxy {
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
