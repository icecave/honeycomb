package cert

import (
	"context"
	"crypto/tls"

	"github.com/icecave/honeycomb/name"
)

// MultiProvider is a provider that combines a slice of providers sequentially.
type MultiProvider struct {
	Providers []Provider
}

// GetCertificate attempts to fetch an existing certificate for the given
// server name. If no such certificate exists, it generates one.
func (m *MultiProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	for _, p := range m.Providers {
		// Look for an existing certificate from the provider.
		if certificate, err := p.GetCertificate(ctx, n); certificate != nil || err != nil {
			return certificate, err
		}
	}

	// finally return nil, nil
	return nil, nil
}

// GetExistingCertificate attempts to fetch an existing certificate for the
// given server name. It never generates new certificates. A non-nil error
// indicates an error with the provider itself; otherwise, a nil certificate
// indicates a failure to find an existing certificate.
func (m *MultiProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	for _, p := range m.Providers {
		// Look for an existing certificate from the provider.
		if certificate, err := p.GetExistingCertificate(ctx, n); certificate != nil || err != nil {
			return certificate, err
		}
	}

	// finally return nil, nil
	return nil, nil
}
