package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

type Certificate struct {
	Addrs        []string
	CertPath     string
	Curve        elliptic.Curve
	KeyPath      string
	Organization string
	Validity     int
}

// NewCertificate creates a new Certificate with default values.
func NewCertificate() *Certificate {
	return &Certificate{
		Addrs:        []string{"127.0.0.1", "localhost"},
		Curve:        elliptic.P256(),
		Organization: "Sentinel",
		Validity:     365,
	}
}

// WithAddrs sets the addresses (IP or DNS) for the certificate.
func (c *Certificate) WithAddrs(addrs []string) *Certificate {
	c.Addrs = addrs
	return c
}

// WithCertPath sets the certificate path.
func (c *Certificate) WithCertPath(certPath string) *Certificate {
	c.CertPath = certPath
	return c
}

// WithCurve sets the elliptic curve for the certificate.
func (c *Certificate) WithCurve(curve elliptic.Curve) *Certificate {
	c.Curve = curve
	return c
}

// WithKeyPath sets the key path.
func (c *Certificate) WithKeyPath(keyPath string) *Certificate {
	c.KeyPath = keyPath
	return c
}

// WithOrganization sets the organization name.
func (c *Certificate) WithOrganization(organization string) *Certificate {
	c.Organization = organization
	return c
}

// WithValidity sets the validity duration for the certificate in days.
func (c *Certificate) WithValidity(days int) *Certificate {
	c.Validity = days
	return c
}

// Generate creates and writes the certificate and private key to the specified paths.
func (c *Certificate) Generate() error {
	// Generate private key
	pk, err := ecdsa.GenerateKey(c.Curve, rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create a random serial number for the certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Separate addresses into domain names and IP addresses
	var domainNames []string
	var ipAddrs []net.IP
	for _, item := range c.Addrs {
		if ip := net.ParseIP(item); ip != nil {
			ipAddrs = append(ipAddrs, ip)
		} else {
			domainNames = append(domainNames, item)
		}
	}

	// Define certificate validity period
	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, c.Validity)

	// Define certificate template
	cert := x509.Certificate{
		BasicConstraintsValid: true,
		DNSNames:              domainNames,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           ipAddrs,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotAfter:              notAfter,
		NotBefore:             notBefore,
		SerialNumber:          serialNumber,
		Subject: pkix.Name{
			Organization: []string{c.Organization},
		},
	}

	// Generate the self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &pk.PublicKey, pk)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write the certificate to file
	if err := utils.WritePEMFile(c.CertPath, "CERTIFICATE", certBytes); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Marshal the private key
	keyBytes, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Write the private key to file
	if err := utils.WritePEMFile(c.KeyPath, "EC PRIVATE KEY", keyBytes); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}
