package cert

import (
	"context"
	"crypto/tls"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/icecave/honeycomb/src/frontend/cert/loader"
	"github.com/icecave/honeycomb/src/name"
)

const certExtension = ".crt"
const keyExtension = ".key"

// LoaderProvider is a certificate provider that reads certificates from a loader.
type LoaderProvider struct {
	Loader loader.Loader

	// Logger is the destination for messages about certificate generation and
	// expiry.
	Logger *log.Logger

	mutex sync.RWMutex
	cache map[string]*tls.Certificate
}

// GetExistingCertificate returns the certificate for the given server name.
func (provider *LoaderProvider) GetExistingCertificate(
	ctx context.Context,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	if tlsCert, ok := provider.findInCache(serverName); ok {
		return tlsCert, nil
	}

	for _, filename := range provider.resolveFilenames(serverName) {
		if cert, err := provider.Loader.LoadCertificate(ctx, filename+certExtension); err == nil {
			if key, err := provider.Loader.LoadPrivateKey(ctx, filename+keyExtension); err == nil {

				if provider.Logger != nil {
					provider.Logger.Printf(
						"Loaded certificate for '%s' from '%s', expires at %s, issued by '%s'",
						serverName.Unicode,
						filename+certExtension,
						cert.NotAfter.Format(time.RFC3339),
						cert.Issuer.CommonName,
					)
				}

				tlsCert := &tls.Certificate{
					Certificate: [][]byte{cert.Raw},
					PrivateKey:  key,
					Leaf:        cert,
				}

				provider.writeToCache(serverName, tlsCert)
				return tlsCert, nil
			}
		}
	}

	return nil, nil
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

func (provider *LoaderProvider) findInCache(
	serverName name.ServerName,
) (*tls.Certificate, bool) {
	provider.mutex.RLock()
	defer provider.mutex.RUnlock()

	cert, ok := provider.cache[serverName.Unicode]

	return cert, ok
}

func (provider *LoaderProvider) writeToCache(
	serverName name.ServerName,
	cert *tls.Certificate,
) {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	if provider.cache == nil {
		provider.cache = map[string]*tls.Certificate{}
	}

	provider.cache[serverName.Unicode] = cert
}
