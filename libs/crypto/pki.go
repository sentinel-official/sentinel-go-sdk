package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sentinel-official/sentinel-go-sdk/libs/encoding/pem"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// PKI manages a public key infrastructure including CA certificate, key, and revocation list.
type PKI struct {
	Dir            string               // Directory where all files (certs, keys, CRLs) are stored.
	Certificate    *x509.Certificate    // Root/CA certificate.
	RevocationList *x509.RevocationList // Current CRL associated with the PKI.
	Signer         crypto.Signer        // Private key used to sign certificates and CRLs.

	m *sync.Mutex // Mutex to guard concurrent updates (e.g., during revocation).
}

// NewPKI creates a new PKI instance bound to the given directory.
func NewPKI(dir string) *PKI {
	return &PKI{
		Dir: dir,
		m:   &sync.Mutex{},
	}
}

// CertPath returns the full file path for a certificate with the given name.
func (p *PKI) CertPath(name string) string {
	return filepath.Join(p.Dir, fmt.Sprintf("%s.crt", name))
}

// KeyPath returns the full file path for a private key with the given name.
func (p *PKI) KeyPath(name string) string {
	return filepath.Join(p.Dir, fmt.Sprintf("%s.key", name))
}

// RLPath returns the full file path for a revocation list with the given name.
func (p *PKI) RLPath(name string) string {
	return filepath.Join(p.Dir, fmt.Sprintf("%s.rl", name))
}

// Init initializes the PKI by generating a root certificate, private key, and CRL.
// Accepts optional certificate customization options via CertOption.
func (p *PKI) Init(opts ...CertOption) (err error) {
	timestamp := time.Now()

	// Create the pki directory if it doesn't exist
	if err := os.MkdirAll(p.Dir, 0755); err != nil {
		return fmt.Errorf("creating PKI directory %q: %w", p.Dir, err)
	}

	// Generate a new ECDSA private key (P-384 curve)
	p.Signer, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating CA private key: %w", err)
	}

	// Marshal the private key to PKCS#8 format
	keyDER, err := x509.MarshalPKCS8PrivateKey(p.Signer)
	if err != nil {
		return fmt.Errorf("marshaling CA private key: %w", err)
	}

	// Define the root certificate template
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "CA",
			Organization: []string{"Sentinel"},
		},
		NotBefore:             timestamp,
		NotAfter:              timestamp.AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	// Apply user-provided customization options
	for _, fn := range opts {
		if err := fn(template); err != nil {
			return fmt.Errorf("applying CA certificate option: %w", err)
		}
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, p.Signer.Public(), p.Signer)
	if err != nil {
		return fmt.Errorf("creating CA certificate: %w", err)
	}

	// Parse the self-signed certificate
	p.Certificate, err = x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("parsing CA certificate: %w", err)
	}

	// Initialize a new empty CRL
	p.RevocationList = &x509.RevocationList{
		SignatureAlgorithm:        p.Certificate.SignatureAlgorithm,
		RevokedCertificateEntries: []x509.RevocationListEntry{},
		Number:                    big.NewInt(1),
		ThisUpdate:                timestamp,
		NextUpdate:                timestamp.AddDate(0, 0, 1), // Valid for 1 day
	}

	// Create and write the CRL to disk
	rlDER, err := x509.CreateRevocationList(rand.Reader, p.RevocationList, p.Certificate, p.Signer)
	if err != nil {
		return fmt.Errorf("creating CA revocation list: %w", err)
	}

	// Persist the CA's private key, certificate and revocation list to disk
	if err := pem.WriteFile(p.KeyPath("ca"), pem.FormatBase64, pem.BlockTypePrivateKey, keyDER); err != nil {
		return fmt.Errorf("writing CA private key: %w", err)
	}
	if err := pem.WriteFile(p.CertPath("ca"), pem.FormatBase64, pem.BlockTypeCertificate, certDER); err != nil {
		return fmt.Errorf("writing CA certificate: %w", err)
	}
	if err := pem.WriteFile(p.RLPath("ca"), pem.FormatBase64, pem.BlockTypeCRL, rlDER); err != nil {
		return fmt.Errorf("writing CA revocation list: %w", err)
	}

	return nil
}

// Issue creates and signs a certificate for a given entity.
// It also writes the generated private key and certificate to disk.
func (p *PKI) Issue(name string, opts ...CertOption) (keyDER []byte, certDER []byte, err error) {
	timestamp := time.Now()

	// Generate a new ECDSA key for the subject
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating private key for %q: %w", name, err)
	}

	// Marshal the private key
	keyDER, err = x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling private key for %q: %w", name, err)
	}

	// Certificate template for the subject
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{"Sentinel"},
		},
		NotBefore:             timestamp,
		NotAfter:              timestamp.AddDate(1, 0, 0), // 1-year validity
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Apply any customization options
	for _, fn := range opts {
		if err := fn(template); err != nil {
			return nil, nil, fmt.Errorf("applying certificate option for %q: %w", name, err)
		}
	}

	// Sign the certificate using the CA's key
	certDER, err = x509.CreateCertificate(rand.Reader, template, p.Certificate, key.Public(), p.Signer)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate for %q: %w", name, err)
	}

	// Write private key and certificate to disk
	if err := pem.WriteFile(p.KeyPath(name), pem.FormatBase64, pem.BlockTypePrivateKey, keyDER); err != nil {
		return nil, nil, fmt.Errorf("writing private key for %q: %w", name, err)
	}
	if err := pem.WriteFile(p.CertPath(name), pem.FormatBase64, pem.BlockTypeCertificate, certDER); err != nil {
		return nil, nil, fmt.Errorf("writing certificate for %q: %w", name, err)
	}

	return keyDER, certDER, nil
}

// Revoke adds a certificate to the revocation list and writes an updated CRL.
func (p *PKI) Revoke(name string) (err error) {
	p.m.Lock()
	defer p.m.Unlock()

	timestamp := time.Now()

	// Load the certificate to be revoked
	cert := new(x509.Certificate)
	if err := pem.ReadFile(p.CertPath(name), pem.FormatBase64, cert); err != nil {
		return fmt.Errorf("reading certificate %q for revocation: %w", name, err)
	}

	// Update CRL number and append the revoked certificate entry
	p.RevocationList.Number = new(big.Int).Add(p.RevocationList.Number, big.NewInt(1))
	p.RevocationList.RevokedCertificateEntries = append(
		p.RevocationList.RevokedCertificateEntries,
		x509.RevocationListEntry{
			SerialNumber:   cert.SerialNumber,
			RevocationTime: timestamp,
			ReasonCode:     9,
		},
	)
	p.RevocationList.ThisUpdate = timestamp
	p.RevocationList.NextUpdate = timestamp.AddDate(0, 0, 1)

	// Create the updated CRL
	rlDER, err := x509.CreateRevocationList(rand.Reader, p.RevocationList, p.Certificate, p.Signer)
	if err != nil {
		return fmt.Errorf("creating updated revocation list: %w", err)
	}

	// Write updated CRL to disk
	if err := pem.WriteFile(p.RLPath("ca"), pem.FormatBase64, pem.BlockTypeCRL, rlDER); err != nil {
		return fmt.Errorf("writing updated revocation list: %w", err)
	}

	// Delete the revoked certificate and key files
	if err := utils.RemoveFile(p.KeyPath(name)); err != nil {
		return fmt.Errorf("removing private key for %q: %w", name, err)
	}
	if err := utils.RemoveFile(p.CertPath(name)); err != nil {
		return fmt.Errorf("removing certificate for %q: %w", name, err)
	}

	return nil
}
