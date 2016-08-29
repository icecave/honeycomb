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
	"time"

	"github.com/lytics/cache"
)

var (
	defaultValidFor     = 24 * time.Hour
	defaultExpiryBuffer = 5 * time.Minute
	defaultCacheSize    = int64(50)
)

// AdhocProvider is a certificate provider that generates new certificates
// when requested. It is used for development domains that are not accessible
// from the internet.
type AdhocProvider struct {
	// The issuer certificate, used to sign new server certificates (typically
	// a self-signed CA certificate).
	IssuerCertificate *tls.Certificate

	// The server's private key, a fixed key is used to allow support for public
	// key pinning across multiple servers.
	ServerKey *rsa.PrivateKey

	// The amount of time that generated certificates are valid for, if zero
	// a value of 24 hours is used.
	ValidFor time.Duration

	// How long before the certificates expire they should be removed from the
	// cache. Certificates are removed before they expire to limit the chance
	// of serving a certificate that will be expired by the time the client
	// validates it. If zero, a value of 5 minutes is used.
	ExpiryBuffer time.Duration

	// The maxium number of certificates to keep in the cache. If zero, a value
	// of 50 is used.
	CacheSize int64

	// Logger specifies an optional logger for messages about certificates
	// generation and cache activity. If nil, logging goes to os.Stderr via the
	// log package's standard logger.
	Logger *log.Logger

	certificates   *cache.Cache
	initializeOnce sync.Once
}

// GetCertificate returns the certificate for the given TLS request. If the
// certificate is not available in the cache, a new one is generated.
func (provider *AdhocProvider) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	provider.initializeOnce.Do(provider.initialize)

	for {
		item, err := provider.certificates.GetOrLoad(strings.ToLower(info.ServerName))
		if err != nil {
			return nil, err
		}

		cert := item.(*tls.Certificate)

		expiresAt := cert.Leaf.NotAfter.Add(-time.Minute)
		if expiresAt.After(time.Now()) {
			return cert, nil
		}

		provider.certificates.ExpireAndHandle(
			provider.CacheSize,
			provider.ValidFor-provider.ExpiryBuffer,
			func(domainName string, _ interface{}) error {
				provider.log("expired certificate for %s", domainName)
				return nil
			},
		)
	}
}

func (provider *AdhocProvider) initialize() {
	if provider.IssuerCertificate.Leaf == nil {
		if len(provider.IssuerCertificate.Certificate) == 0 {
			panic("no X509 certificates in issuer certificate")
		}
		var err error
		provider.IssuerCertificate.Leaf, err = x509.ParseCertificate(
			provider.IssuerCertificate.Certificate[0],
		)
		if err != nil {
			panic(err)
		}
	}

	if provider.ValidFor == 0 {
		provider.ValidFor = defaultValidFor
	}
	if provider.ExpiryBuffer == 0 {
		provider.ExpiryBuffer = defaultExpiryBuffer
	}
	if provider.CacheSize == 0 {
		provider.CacheSize = defaultCacheSize
	}

	provider.certificates = cache.NewCache(
		8, // number of stripes (separate locks)
		provider.load,
		cache.SizerAlwaysOne,
	)

	provider.log(
		"initialized ad-hoc certificate provider, issuer: %s, valid-for: %s, expiry-buffer: %s, cache-size: %d",
		provider.IssuerCertificate.Leaf.Subject.CommonName,
		provider.ValidFor,
		provider.ExpiryBuffer,
		provider.CacheSize,
	)
}

func (provider *AdhocProvider) load(domainName string) (interface{}, error) {
	template, err := provider.newTemplate(domainName)
	if err != nil {
		return nil, err
	}

	raw, err := x509.CreateCertificate(
		rand.Reader,
		template,
		provider.IssuerCertificate.Leaf,
		&provider.ServerKey.PublicKey,
		provider.IssuerCertificate.PrivateKey,
	)
	if err != nil {
		return nil, err
	}

	certificate, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, err
	}

	provider.log("generated certificate for %s", domainName)

	return &tls.Certificate{
		Certificate: [][]byte{raw},
		PrivateKey:  provider.ServerKey,
		Leaf:        certificate,
	}, nil
}

func (provider *AdhocProvider) newTemplate(domainName string) (*x509.Certificate, error) {
	serialNumber, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(provider.ValidFor)

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

func (provider *AdhocProvider) log(format string, args ...interface{}) {
	if provider.Logger != nil {
		provider.Logger.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
