package cert

import (
	"context"
	"crypto/tls"

	"github.com/icecave/honeycomb/src/name"
)

// Provider fetches TLS certificates for incoming TLS requests.
type Provider interface {
	// GetExistingCertificate returns the certificate for the given server name,
	// if it has already been generated. If the certificate has not been
	// generated the returned certificate and error are both nil.
	GetExistingCertificate(ctx context.Context, serverName name.ServerName) (*tls.Certificate, error)

	// GetCertificate returns the certificate for the given server name. If the
	// certificate doe not exist, it attempts to generate one.
	GetCertificate(ctx context.Context, serverName name.ServerName) (*tls.Certificate, error)
}
