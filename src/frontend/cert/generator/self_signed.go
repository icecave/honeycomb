package generator

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"time"
)

// SelfSignedGenerator generates new self-signed server certificates.
type SelfSignedGenerator struct {
	// ServerKey is the server's private RSA key.
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
func (generator *SelfSignedGenerator) Generate(
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
		template,
		&generator.ServerKey.PublicKey,
		generator.ServerKey,
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
