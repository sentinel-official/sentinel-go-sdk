package crypto

import (
	"crypto/x509"
	"net"
	"net/mail"
	"net/url"
)

// CertOption defines a functional option for modifying a certificate template.
// These are applied during certificate creation to customize properties.
type CertOption func(c *x509.Certificate) error

// CertSAN returns a CertOption that appends various types of Subject Alternative Names
// to the certificate, such as IPs, emails, URIs, and DNS names.
func CertSAN(sans ...string) CertOption {
	return func(c *x509.Certificate) error {
		for _, san := range sans {
			if ip := net.ParseIP(san); ip != nil {
				c.IPAddresses = append(c.IPAddresses, ip)
			} else if email, err := mail.ParseAddress(san); err == nil {
				c.EmailAddresses = append(c.EmailAddresses, email.Address)
			} else if uri, err := url.ParseRequestURI(san); err == nil {
				c.URIs = append(c.URIs, uri)
			} else {
				c.DNSNames = append(c.DNSNames, san)
			}
		}

		return nil
	}
}
