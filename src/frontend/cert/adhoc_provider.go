package cert

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

// NewAdhocProvider returns a certificate provider that generates new
// certificates when requested. It is used for development servers that are not
// accessible from the internet.
func NewAdhocProvider(
	issuerCertificate *tls.Certificate,
	serverKey *rsa.PrivateKey,
	validFor time.Duration,
	expiryBuffer time.Duration,
	logger *log.Logger,
) Provider {
	if issuerCertificate.Leaf == nil {
		var err error
		issuerCertificate.Leaf, err = x509.ParseCertificate(issuerCertificate.Certificate[0])
		if err != nil {
			panic(err)
		}
	}

	provider := &adhocProvider{
		issuerCertificate: issuerCertificate,
		serverKey:         serverKey,
		validFor:          validFor,
		expiryBuffer:      expiryBuffer,
		logger:            logger,
	}

	provider.cache.Store(certificateCache{})

	return provider
}

// adhocProvider is a certificate provider that generates new certificates
// when requested. It is used for development servers that are not accessible
// from the internet.
type adhocProvider struct {
	// The issuer certificate, used to sign new server certificates (typically
	// a self-signed CA certificate).
	issuerCertificate *tls.Certificate

	// The server's private key, a fixed key is used to allow support for public
	// key pinning across multiple servers.
	serverKey *rsa.PrivateKey

	// The amount of time that generated certificates are valid for.
	validFor time.Duration

	// The amount of time BEFORE a certificate it expires that it should be
	// removed from the cache. Certificates are removed before they expire to
	// limit the chance of serving a certificate that will be expired by the
	// time the client validates it. If zero, a value of 5 minutes is used.
	expiryBuffer time.Duration

	// Logger specifies a logger for messages about certificate generation and
	// cache expiration.
	logger *log.Logger

	// The certificate cache, maps server name to TLS certificate.
	cache atomic.Value

	// A mutex for ensuring only one certificate is generated at a time.
	mutex sync.Mutex
}

type certificateCache map[string]*tls.Certificate

// GetExistingCertificate returns the certificate for the given server name,
// if it has already been generated. If the certificate has not been
// generated the returned certificate and error are both nil.
func (provider *adhocProvider) GetExistingCertificate(_ context.Context, serverName name.ServerName) (*tls.Certificate, error) {
	// Load the cache object atomically ...
	cache := provider.cache.Load().(certificateCache)
	certificate := cache[serverName.Unicode]

	// Return nothing if the certificate is not found or otherwise expired ...
	if certificate == nil || provider.isStale(certificate) {
		return nil, nil
	}

	// Return the certificate if it's present and not expired ...
	return certificate, nil
}

// GetCertificate returns the certificate for the given server name. If the
// certificate doe not exist, it attempts to generate one.
func (provider *adhocProvider) GetCertificate(ctx context.Context, serverName name.ServerName) (*tls.Certificate, error) {
	certificate, err := provider.GetExistingCertificate(ctx, serverName)
	if err != nil {
		return nil, err
	} else if certificate != nil {
		return certificate, nil
	}

	return provider.generate(serverName)
}

// generate creates a new certificate for the given server name and stores it
// in the cache, it also purges any expired certificates.
func (provider *adhocProvider) generate(serverName name.ServerName) (*tls.Certificate, error) {
	// Acquire the mutex, preventing two goroutines from possibly generating the
	// same certificate. Lock contention *may* be improved by using a seperate
	// mutex for each server, but this is probably fine for development ...
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	// Check the cache in case another goroutine has generated the certificate
	// while we were waiting for the mutex ...
	cache := provider.cache.Load().(certificateCache)
	certificate := cache[serverName.Unicode]
	if certificate != nil && !provider.isStale(certificate) {
		return certificate, nil
	}

	// Create the new certificate ...
	certificate, err := provider.newCertificate(serverName)
	if err != nil {
		return nil, err
	}

	// Create a clone of the cache without expired certificates, and with the
	// newly generated certificate ...
	clone := provider.purge(cache)
	clone[serverName.Unicode] = certificate

	// Atomically replace the cache ...
	provider.cache.Store(clone)

	provider.logger.Printf(
		"frontend: Issued certificate for '%s', expires at %s, issued by '%s'",
		serverName.Unicode,
		certificate.Leaf.NotAfter.Format(time.RFC3339),
		certificate.Leaf.Issuer.CommonName,
	)

	return certificate, nil
}

// purge creates a copy of the given certificate cache excluding any stale
// certificates ...
func (provider *adhocProvider) purge(cache certificateCache) certificateCache {
	clone := certificateCache{}
	for serverNameUnicode, certificate := range cache {
		if provider.isStale(certificate) {
			provider.logger.Printf(
				"frontend: Expired certificate for '%s', expired at %s",
				serverNameUnicode,
				certificate.Leaf.NotAfter.Format(time.RFC3339),
			)
		} else {
			clone[serverNameUnicode] = certificate
		}
	}

	return clone
}

// isStale returns true if the given certificate should be removed from the
// cache.
func (provider *adhocProvider) isStale(certificate *tls.Certificate) bool {
	expiresAt := certificate.Leaf.NotAfter.Add(-provider.expiryBuffer)
	return time.Now().After(expiresAt)
}

// newCertificate generates a new TLS certificate for the given server name.
func (provider *adhocProvider) newCertificate(serverName name.ServerName) (*tls.Certificate, error) {
	template, err := provider.newTemplate(serverName)
	if err != nil {
		return nil, err
	}

	raw, err := x509.CreateCertificate(
		rand.Reader,
		template,
		provider.issuerCertificate.Leaf,
		&provider.serverKey.PublicKey,
		provider.issuerCertificate.PrivateKey,
	)
	if err != nil {
		return nil, err
	}

	certificate, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{raw},
		PrivateKey:  provider.serverKey,
		Leaf:        certificate,
	}, nil
}

// newTemplate returns the certificate template used to make new certificates.
func (provider *adhocProvider) newTemplate(serverName name.ServerName) (*x509.Certificate, error) {
	serialNumber, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(provider.validFor)

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: serverName.Unicode},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{serverName.Punycode},
	}, nil
}
