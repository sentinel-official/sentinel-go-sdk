package proxycmd

// VLESSAccount encodes a VLESS account value message with fields id (1) and flow (2).
// Empty strings are omitted per proto3 default-omission.
func VLESSAccount(id, flow string) []byte {
	var b []byte

	if id != "" {
		b = appendString(b, 1, id)
	}

	if flow != "" {
		b = appendString(b, 2, flow)
	}

	return b
}

// VMessAccount encodes a VMess account value message with field id (1).
// An empty id is omitted per proto3 default-omission.
func VMessAccount(id string) []byte {
	var b []byte

	if id != "" {
		b = appendString(b, 1, id)
	}

	return b
}

// TrojanAccount encodes a Trojan account value message with field password (1).
// An empty password is omitted per proto3 default-omission.
func TrojanAccount(password string) []byte {
	var b []byte

	if password != "" {
		b = appendString(b, 1, password)
	}

	return b
}

// ShadowsocksAccount encodes a Shadowsocks 2022 account value message with field key (1).
// An empty key is omitted per proto3 default-omission.
func ShadowsocksAccount(key string) []byte {
	var b []byte

	if key != "" {
		b = appendString(b, 1, key)
	}

	return b
}
