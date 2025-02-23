package v2ray

// ServerMetadata represents metadata for a V2Ray server's inbound connection.
type ServerMetadata struct {
	Tag *Tag `json:"tag"` // Tag uniquely identifies the inbound connection.
}
