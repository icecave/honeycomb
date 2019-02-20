package cert

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

// DefaultTimeout specifies the duration to allow for fetching a certificate if
// no other timeout is specified.
const DefaultTimeout = 5 * time.Second

// ProviderAdaptor wraps a primary and secondary Provider to present an
// interface suitable for use as the tls.Config "GetEnvironment" callback.
type ProviderAdaptor struct {
	// PrimaryProvider is the certificate provider used to create certificates
	// for "normal" recognized server names.
	PrimaryProvider Provider

	// SecondaryProvider is used to provide default or "fallback" certificates
	// so that requests may be served even when the server name is unrecognized.
	SecondaryProvider Provider

	// Timeout is the maximum time allowed for a certificate request to complete.
	// If the timeout is zero, the value of DefaultTimeout is used.
	Timeout time.Duration

	// IsRecognised is a predicate function that is used to work out which
	// certificate provider to use for a given server name. If IsRecognised is
	// nil, all server names are considered unrecognized.
	IsRecognised func(context.Context, name.ServerName) bool
}

// GetCertificate forwards certificate requests to the appropriate provider.
func (adaptor *ProviderAdaptor) GetCertificate(
	info *tls.ClientHelloInfo,
) (*tls.Certificate, error) {
	ctx, cancel := adaptor.context()
	defer cancel()

	// Attempt to parse the server name. If it's missing or invalid, use the
	// default certificate from the secondary provider. This way we can at least
	// show an HTTP error message to the user ...
	serverName, err := name.FromTLS(info)
	if err != nil {
		return nil, nil
	}

	// Next, look for an existing certificate from the primary provider. If such
	// a certificate is available, it doesn't matter if the server name is
	// recognized or not ...
	certificate, err := adaptor.PrimaryProvider.GetExistingCertificate(ctx, serverName)
	if certificate != nil || err != nil {
		return certificate, err
	}

	// If the server name is recognized, use the primary provider to get a new
	// certificate for the server name ...
	if adaptor.IsRecognised != nil && adaptor.IsRecognised(ctx, serverName) {
		return adaptor.PrimaryProvider.GetCertificate(ctx, serverName)
	}

	// Finally, fallback to the secondary provider ...
	return adaptor.SecondaryProvider.GetCertificate(ctx, serverName)
}

// context returns a new context to use for a request.
func (adaptor *ProviderAdaptor) context() (context.Context, context.CancelFunc) {
	timeout := adaptor.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return context.WithTimeout(context.Background(), timeout)
}
