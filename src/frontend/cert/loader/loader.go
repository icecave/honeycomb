package loader

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
)

// Loader loads certificates and keys from file names.
type Loader interface {
	// LoadCertificate reads an x509 certificate from a file in PEM format and
	// returns the parsed certificate.
	LoadCertificate(ctx context.Context, certFile string) (*x509.Certificate, error)

	// LoadPrivateKey reads an RSA certificate from a file in PEM formt and
	// returns the parsed key.
	LoadPrivateKey(ctx context.Context, keyFile string) (*rsa.PrivateKey, error)
}

// LoadX509KeyPair uses the specified laoder to create a cert/key pair.
func LoadX509KeyPair(
	ctx context.Context,
	loader Loader,
	certFile string,
	keyFile string,
) (*tls.Certificate, error) {
	x509, err := loader.LoadCertificate(ctx, certFile)
	if err != nil {
		return nil, err
	}

	key, err := loader.LoadPrivateKey(ctx, keyFile)
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, x509.Raw)
	cert.PrivateKey = key
	cert.Leaf = x509

	return &cert, nil
}
