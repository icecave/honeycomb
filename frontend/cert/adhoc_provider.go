package cert

import (
	"context"
	"crypto/tls"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icecave/honeycomb/frontend/cert/generator"
	"github.com/icecave/honeycomb/name"
)

// DefaultTTLOffset is the default amount of time before a certificate expires
// that it is removed from the cache..
const DefaultTTLOffset = -15 * time.Minute

// AdhocProvider is a certificate provider that creates new certificates on the
// fly using a certificate generator.
type AdhocProvider struct {
	// Generator is the certificate generator used to create new certificates.
	Generator generator.Generator

	// TTLOffset is the amount of time before a certificate expires that it is
	// removed from the cache. This is done to prevent serving a certificate
	// that is about to expire to a client, and to account for some clock-drift
	// between server and client.
	TTLOffset time.Duration

	// Logger is the destination for messages about certificate generation and
	// expiry.
	Logger *log.Logger

	cache atomic.Value // certificateCache or nil
	mutex sync.Mutex
}

// GetExistingCertificate returns the certificate for the given server name,
// if it has already been generated. If the certificate has not been
// generated the returned certificate and error are both nil.
func (provider *AdhocProvider) GetExistingCertificate(
	_ context.Context,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	cache, _ := provider.cache.Load().(certificateCache)
	return provider.fetch(cache, serverName), nil
}

// GetCertificate returns the certificate for the given server name. If the
// certificate doe not exist, it attempts to generate one.
func (provider *AdhocProvider) GetCertificate(
	ctx context.Context,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	cache, _ := provider.cache.Load().(certificateCache)
	if certificate := provider.fetch(cache, serverName); certificate != nil {
		return certificate, nil
	}

	return provider.generate(
		ctx,
		serverName.Unicode,
		serverName,
	)
}

func (provider *AdhocProvider) generate(
	ctx context.Context,
	commonName string,
	serverName name.ServerName,
) (*tls.Certificate, error) {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	cache, _ := provider.cache.Load().(certificateCache)
	if certificate := provider.fetch(cache, serverName); certificate != nil {
		return certificate, nil
	}

	cache = provider.purge(cache)

	certificate, err := provider.Generator.Generate(
		ctx,
		commonName,
		serverName.Punycode,
	)
	if err != nil {
		return nil, err
	}

	cache[serverName.Unicode] = certificate
	provider.cache.Store(cache)

	if provider.Logger != nil {
		provider.Logger.Printf(
			"Issued certificate for '%s', expires at %s, issued by '%s'",
			serverName.Unicode,
			certificate.Leaf.NotAfter.Format(time.RFC3339),
			certificate.Leaf.Issuer.CommonName,
		)
	}

	return certificate, nil
}

// purge returns a new cache that does not contain any stale certificates.
func (provider *AdhocProvider) purge(cache certificateCache) certificateCache {
	result := certificateCache{}

	for unicodeServerName, certificate := range cache {
		if !provider.isStale(certificate) {
			result[unicodeServerName] = certificate
		} else if provider.Logger != nil {
			provider.Logger.Printf(
				"Expired certificate for '%s', expired at %s",
				unicodeServerName,
				certificate.Leaf.NotAfter.Format(time.RFC3339),
			)
		}
	}

	return result
}

// isStale checks if the given certificate should be removed from the cache.
func (provider *AdhocProvider) isStale(certificate *tls.Certificate) bool {
	ttlOffset := provider.TTLOffset
	if ttlOffset == 0 {
		ttlOffset = DefaultTTLOffset
	}

	expiresAt := certificate.Leaf.NotAfter.Add(ttlOffset)

	return time.Now().After(expiresAt)
}

// fetch returns an existing certificate from a cache object, if present.
func (provider *AdhocProvider) fetch(
	cache certificateCache,
	serverName name.ServerName,
) *tls.Certificate {
	if certificate, ok := cache[serverName.Unicode]; ok {
		if !provider.isStale(certificate) {
			return certificate
		}
	}

	return nil
}

type certificateCache map[string]*tls.Certificate
