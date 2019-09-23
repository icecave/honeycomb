package cert

import (
	"crypto/tls"
)

// Provider fetches or creates TLS certificates for incoming HTTPS requests.
type Provider interface {
	// GetCertificate attempts to fetch a certificate for the given request.
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}
