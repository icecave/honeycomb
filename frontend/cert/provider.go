package cert

import (
	"context"
	"crypto/tls"

	"github.com/icecave/honeycomb/name"
)

// Provider fetches or creates TLS certificates for incoming HTTPS requests.
type Provider interface {
	// GetCertificate attempts to fetch an existing certificate for the given
	// server name. If no such certificate exists, it generates one.
	GetCertificate(context.Context, name.ServerName) (*tls.Certificate, error)

	// GetExistingCertificate attempts to fetch an existing certificate for the
	// given server name. It never generates new certificates. A non-nil error
	// indicates an error with the provider itself; otherwise, a nil certificate
	// indicates a failure to find an existing certificate.
	GetExistingCertificate(context.Context, name.ServerName) (*tls.Certificate, error)
}
