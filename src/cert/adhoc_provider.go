package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// NewAdhocProvider returns a certificate provider that generates new
// certificates when requested. It is used for development domains that are not
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
// when requested. It is used for development domains that are not accessible
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

	// The certificate cache, maps domain name to TLS certificate.
	cache atomic.Value

	// A mutex for ensuring only one certificate is generated at a time.
	mutex sync.Mutex
}

type certificateCache map[string]*tls.Certificate

// GetCertificate returns the certificate for the given TLS request. If the
// certificate is not available in the cache, a new one is generated.
func (provider *adhocProvider) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Normalize the domain name ...
	domainName := strings.ToLower(info.ServerName)

	// Load the cache object atomically ...
	cache := provider.cache.Load().(certificateCache)
	certificate := cache[domainName]

	// Return the certificate if it's present and not expired ...
	if certificate != nil && !provider.isStale(certificate) {
		return certificate, nil
	}

	// Generate a new certificate ...
	return provider.generate(domainName)
}

// generate creates a new certificate for the given domain name and stores it
// in the cache, it also purges any expired certificates.
func (provider *adhocProvider) generate(domainName string) (*tls.Certificate, error) {
	// Acquire the mutex, preventing two goroutines from possibly generating the
	// same certificate. Lock contention *may* be improved by using a seperate
	// mutex for each domain, but this is probably fine for development ...
	provider.mutex.Lock()
	defer provider.mutex.Unlock()

	// Check the cache in case another goroutine has generated the certificate
	// while we were waiting for the mutex ...
	cache := provider.cache.Load().(certificateCache)
	certificate := cache[domainName]
	if certificate != nil && !provider.isStale(certificate) {
		return certificate, nil
	}

	// Create the new certificate ...
	certificate, err := provider.newCertificate(domainName)
	if err != nil {
		return nil, err
	}

	// Create a clone of the cache without expired certificates, and with the
	// newly generated certificate ...
	clone := provider.purge(cache)
	clone[domainName] = certificate

	// Atomically replace the cache ...
	provider.cache.Store(clone)

	provider.logger.Printf(
		"cert: Issued certificate for '%s', expires at %s, issued by '%s'",
		domainName,
		certificate.Leaf.NotAfter.Format(time.RFC3339),
		certificate.Leaf.Issuer.CommonName,
	)

	return certificate, nil
}

// purge creates a copy of the given certificate cache excluding any stale
// certificates ...
func (provider *adhocProvider) purge(cache certificateCache) certificateCache {
	clone := certificateCache{}
	for domainName, certificate := range cache {
		if provider.isStale(certificate) {
			provider.logger.Printf(
				"cert: Expired certificate for '%s', expired at %s",
				domainName,
				certificate.Leaf.NotAfter.Format(time.RFC3339),
			)
		} else {
			clone[domainName] = certificate
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

// newCertificate generates a new TLS certificate for the given domain name.
func (provider *adhocProvider) newCertificate(domainName string) (*tls.Certificate, error) {
	template, err := provider.newTemplate(domainName)
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
func (provider *adhocProvider) newTemplate(domainName string) (*x509.Certificate, error) {
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
		Subject:               pkix.Name{CommonName: domainName},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{domainName},
	}, nil
}
