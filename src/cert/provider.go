package cert

import "crypto/tls"

// Provider fetches TLS certificates for incoming TLS requests.
type Provider interface {
	// GetExistingCertificate returns the certificate for the given server name,
	// if it has already been generated. If the certificate has not been
	// generated the returned certificate and error are both nil.
	GetExistingCertificate(serverName string) (*tls.Certificate, error)

	// GetCertificate returns the certificate for the given server name. If the
	// certificate doe not exist, it attempts to generate one.
	GetCertificate(serverName string) (*tls.Certificate, error)
}
