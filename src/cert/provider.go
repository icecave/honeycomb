package cert

import "crypto/tls"

// Provider fetches TLS certificates for incoming TLS requests.
type Provider interface {
	GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error)
}
