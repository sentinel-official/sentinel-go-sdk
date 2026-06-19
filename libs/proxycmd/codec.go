package proxycmd

// rawCodec is a gRPC codec that passes bytes through without modification.
// Name() returns "proto" because ForceCodec sets the gRPC content-subtype to
// the codec name, and the xray/v2ray servers only accept the "proto" subtype.
// ForceCodec is applied per-call and does NOT replace the global codec, so
// returning "proto" here is safe and required; do not change it to a custom name.
type rawCodec struct{}

func (rawCodec) Marshal(v any) ([]byte, error) {
	return v.([]byte), nil
}

func (rawCodec) Unmarshal(data []byte, v any) error {
	*(v.(*[]byte)) = data

	return nil
}

// Name returns "proto" so that gRPC sets the correct content-subtype for xray/v2ray servers.
func (rawCodec) Name() string {
	return "proto"
}
