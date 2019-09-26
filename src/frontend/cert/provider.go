package cert

import (
	"crypto/tls"

	"github.com/icecave/honeycomb/src/name"
)

// Provider is an interface for obtaining certificates for server names.
type Provider interface {
	// GetCertificate returns the server certificate for a specific server name,
	// if available.
	GetCertificate(name.ServerName, *tls.ClientHelloInfo) (ProviderResult, bool)

	// IsValid returns true a cached provider result should still be considered
	// valid.
	//
	// The behavior is undefined if the result was not obtained from this
	// provider.
	IsValid(ProviderResult) bool
}

// ProviderResult is the result of asking a provider for a certificate.
type ProviderResult struct {
	// Certificate is the certificate itself.
	Certificate *tls.Certificate

	// ExcludeFromCache is true if this result should never be cached.
	ExcludeFromCache bool
}
