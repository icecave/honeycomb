package cert

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"

	"github.com/icecave/honeycomb/src/frontend/cert/loader"
	"github.com/icecave/honeycomb/src/name"
)

const certExtension = ".crt"
const keyExtension = ".key"

// LoaderProvider is a certificate provider that reads certificates from a loader.
type LoaderProvider struct {
	Loader loader.Loader
}

// GetExistingCertificate returns the certificate for the given server name.
func (provider *LoaderProvider) GetExistingCertificate(
	ctx context.Context,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	for _, filename := range provider.resolveFilenames(serverName) {
		if cert, err := provider.Loader.LoadCertificate(ctx, filename+certExtension); err != nil {
			if key, err := provider.Loader.LoadPrivateKey(ctx, filename+keyExtension); err != nil {
				return &tls.Certificate{
					Certificate: [][]byte{cert.Raw},
					PrivateKey:  key,
					Leaf:        cert,
				}, nil
			}
		}
	}

	return nil, errors.New("certificate not found")
}

// GetCertificate returns the certificate for the given server name. If the
// certificate has not been loading, it attempts to load one.
func (provider *LoaderProvider) GetCertificate(
	ctx context.Context,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	return provider.GetExistingCertificate(ctx, serverName)
}

func (provider *LoaderProvider) resolveFilenames(
	serverName name.ServerName,
) []string {
	split := strings.SplitN(serverName.Punycode, ".", 2)
	if len(split) == 1 {
		return []string{
			serverName.Punycode,
		}
	}

	return []string{
		serverName.Punycode,
		"_." + split[1],
		"_." + serverName.Punycode,
	}
}
