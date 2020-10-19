package cert

import (
	"context"
	"crypto/tls"

	"github.com/icecave/honeycomb/name"
)

// MultiProvider is a provider that combines two providers sequentially, this can be used recursively to
// add more than two Providers together.
type MultiProvider struct {
	PrimaryProvider   Provider
	SecondaryProvider Provider
}

// GetCertificate attempts to fetch an existing certificate for the given
// server name. If no such certificate exists, it generates one.
func (m *MultiProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	// Look for an existing certificate from the primary provider. If such
	// a certificate is available, it doesn't matter if the server name is
	// recognized or not ...
	certificate, err := m.PrimaryProvider.GetCertificate(ctx, n)
	if certificate != nil || err != nil {
		return certificate, err
	}

	// Finally, fallback to the secondary provider ...
	return m.SecondaryProvider.GetCertificate(ctx, n)
}

// GetExistingCertificate attempts to fetch an existing certificate for the
// given server name. It never generates new certificates. A non-nil error
// indicates an error with the provider itself; otherwise, a nil certificate
// indicates a failure to find an existing certificate.
func (m *MultiProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	// Look for an existing certificate from the primary provider. If such
	// a certificate is available, it doesn't matter if the server name is
	// recognized or not ...
	certificate, err := m.PrimaryProvider.GetExistingCertificate(ctx, n)
	if certificate != nil || err != nil {
		return certificate, err
	}

	// Finally, fallback to the secondary provider ...
	return m.SecondaryProvider.GetExistingCertificate(ctx, n)
}
