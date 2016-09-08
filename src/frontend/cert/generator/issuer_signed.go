package generator

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"time"
)

// IssuerSignedGenerator generates new server certificates signed by a separate
// issuer certificate.
type IssuerSignedGenerator struct {
	// IssuerCertificate is the X509 certificate of the issuer, typically a
	// sign-signed CA certificate.
	IssuerCertificate *x509.Certificate

	// IssuerKey is the issuer's private key.
	IssuerKey crypto.PrivateKey

	// ServerKey is the server's private RSA key.
	// @todo support other key types
	ServerKey *rsa.PrivateKey

	// NotBeforeOffset specifies the amount of time added to the current time to
	// produce the "NotBefore" value for a new certificate. It is typically
	// negative to allow for some clock-drift between client and server. If the
	// value is zero, DefaultNotBeforeOffset is used.
	NotBeforeOffset time.Duration

	// NotAfterOffset specifies the amount of time added to the current time to
	// produce the "NotAfter" value for a new certificate. If the value is zero,
	// DefaultNotAfterOffset is used.
	NotAfterOffset time.Duration
}

// Generate creates a new TLS certificate for the given server name.
func (generator *IssuerSignedGenerator) Generate(
	ctx context.Context,
	commonName string,
	dnsName string,
) (*tls.Certificate, error) {
	template, err := newTemplateCertificate(
		commonName,
		dnsName,
		generator.NotBeforeOffset,
		generator.NotAfterOffset,
	)
	if err != nil {
		return nil, err
	}

	raw, err := x509.CreateCertificate(
		rand.Reader,
		template,
		generator.IssuerCertificate,
		&generator.ServerKey.PublicKey,
		generator.IssuerKey,
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
		PrivateKey:  generator.ServerKey,
		Leaf:        certificate,
	}, nil
}
