package pem

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
)

// PEM block type constants for various supported data types.
const (
	BlockTypeCertificate        = "CERTIFICATE"
	BlockTypePrivateKey         = "PRIVATE KEY"
	BlockTypeCRL                = "X509 CRL"
	BlockTypeOpenVPNStaticKeyV1 = "OpenVPN Static key V1"
)

// Format represents an encoding format (e.g., Base64 or Hex).
type Format string

// Supported encoding formats.
const (
	FormatBase64 Format = "base64"
	FormatHex    Format = "hex"
)

// Constants for hex encoding formatting.
const (
	hexLineLen  = 32            // Number of hex characters per line
	beginMarker = "-----BEGIN " // Start marker of a PEM block
	endMarker   = "-----END "   // End marker of a PEM block
)

// Encode encodes a PEM block into the specified format and writes it to the provided writer.
func Encode(w io.Writer, b *pem.Block, format Format) error {
	switch format {
	case FormatBase64:
		return encodeBase64(w, b)
	case FormatHex:
		return encodeHex(w, b)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

// Decode decodes data from the specified format and returns a PEM block and any leftover data.
func Decode(data []byte, format Format) (*pem.Block, []byte) {
	switch format {
	case FormatBase64:
		return decodeBase64(data)
	case FormatHex:
		return decodeHex(data)
	default:
		return nil, data
	}
}

// WriteFile writes encoded data as a PEM block to the specified file.
func WriteFile(path string, format Format, blockType string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", path, err)
	}

	defer func() {
		_ = file.Close()
	}()

	block := &pem.Block{Type: blockType, Bytes: data}
	if err := Encode(file, block, format); err != nil {
		return fmt.Errorf("encoding %s block to %q: %w", blockType, path, err)
	}

	return nil
}

// ReadFile reads a PEM-encoded file, decodes it, and parses the content into the provided output object.
func ReadFile(path string, format Format, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file %q: %w", path, err)
	}

	block, _ := Decode(data, format)
	if block == nil {
		return fmt.Errorf("decoding PEM block from %q: got nil block", path)
	}

	switch block.Type {
	case BlockTypeCertificate:
		parsed, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("parsing certificate: %w", err)
		}

		ptr, ok := out.(*x509.Certificate)
		if !ok {
			return fmt.Errorf("output parameter for %q is %T, expected *x509.Certificate", path, out)
		}

		*ptr = *parsed
	case BlockTypePrivateKey:
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parsing private key: %w", err)
		}

		switch pk := parsed.(type) {
		case *ecdsa.PrivateKey:
			ptr, ok := out.(*ecdsa.PrivateKey)
			if !ok {
				return fmt.Errorf("output parameter for %q is %T, expected *ecdsa.PrivateKey", path, out)
			}

			*ptr = *pk
		default:
			return fmt.Errorf("unsupported private-key subtype %T in %q", pk, path)
		}
	case BlockTypeCRL:
		parsed, err := x509.ParseRevocationList(block.Bytes)
		if err != nil {
			return fmt.Errorf("parsing revocation list: %w", err)
		}

		ptr, ok := out.(*x509.RevocationList)
		if !ok {
			return fmt.Errorf("output parameter for %q is %T, expected *x509.RevocationList", path, out)
		}

		*ptr = *parsed
	default:
		return fmt.Errorf("unsupported PEM block type %q in %q", block.Type, path)
	}

	return nil
}

// encodeBase64 encodes and writes a PEM block using standard base64 encoding.
func encodeBase64(w io.Writer, b *pem.Block) error {
	return pem.Encode(w, b)
}

// decodeBase64 decodes the first base64 PEM block found in the data.
func decodeBase64(data []byte) (*pem.Block, []byte) {
	return pem.Decode(data)
}

// encodeHex encodes a PEM block using hex and writes it with proper headers and line breaks.
func encodeHex(w io.Writer, b *pem.Block) error {
	// Write BEGIN header
	if _, err := fmt.Fprintf(w, "%s%s-----\n", beginMarker, b.Type); err != nil {
		return err
	}

	// Encode block bytes to hex
	hexData := make([]byte, hex.EncodedLen(len(b.Bytes)))
	hex.Encode(hexData, b.Bytes)

	// Write hex data in lines
	for i := 0; i < len(hexData); i += hexLineLen {
		end := i + hexLineLen
		if end > len(hexData) {
			end = len(hexData)
		}

		if _, err := w.Write(hexData[i:end]); err != nil {
			return err
		}

		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}

	// Write END footer
	if _, err := fmt.Fprintf(w, "%s%s-----\n", endMarker, b.Type); err != nil {
		return err
	}

	return nil
}

// decodeHex parses and decodes a PEM block encoded in hex format.
func decodeHex(data []byte) (*pem.Block, []byte) {
	// Find the beginning marker
	idx := bytes.Index(data, []byte(beginMarker))
	if idx < 0 {
		return nil, data
	}

	// Find the end marker
	endIdx := bytes.Index(data[idx:], []byte(endMarker))
	if endIdx < 0 {
		return nil, data
	}

	remainder := data[idx+endIdx:]

	newline := bytes.IndexByte(remainder, '\n')
	if newline >= 0 {
		remainder = remainder[newline+1:]
	}

	// Extract block lines between BEGIN and END
	blockData := data[idx : idx+endIdx]

	lines := bytes.Split(blockData, []byte{'\n'})
	if len(lines) < 2 {
		return nil, data
	}

	// Parse type from BEGIN line
	typeLine := strings.TrimSuffix(
		strings.TrimPrefix(string(lines[0]), beginMarker), "-----",
	)

	// Gather all hex data lines, excluding END marker
	var hexBuffer []byte
	for _, line := range lines[1:] {
		if bytes.HasPrefix(line, []byte(endMarker)) {
			break
		}

		l := bytes.TrimSpace(bytes.ReplaceAll(line, []byte("\r"), nil))
		if len(l) > 0 {
			hexBuffer = append(hexBuffer, l...)
		}
	}

	// Decode hex data into raw bytes
	decoded := make([]byte, hex.DecodedLen(len(hexBuffer)))

	n, err := hex.Decode(decoded, hexBuffer)
	if err != nil {
		return nil, data
	}

	block := &pem.Block{Type: typeLine, Bytes: decoded[:n]}

	return block, remainder
}
