package cert

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"github.com/icecave/honeycomb/src/name"
)

// AdhocProvider is a certificate provider that creates new certificates on the
// fly using a certificate generator.
//
// It should always be the lowest priority provider as (barring some problem
// with the generator itself) it will always provide a certificate.
type AdhocProvider struct {
	// Generator is the certificate generator used to create new certificates.
	Generator generator.Generator

	// Logger is the destination for messages about certificate generation.
	Logger *log.Logger
}

// GetCertificate returns a generated certificate for the given server name.
func (p *AdhocProvider) GetCertificate(
	n name.ServerName,
	_ *tls.ClientHelloInfo,
) (ProviderResult, bool) {
	c, err := p.Generator.Generate(
		context.Background(),
		n.Unicode,
		n.Punycode,
	)
	if err != nil {
		p.Logger.Printf(
			"Unable to issue certificate for '%s': %s",
			n.Unicode,
			err,
		)

		return ProviderResult{}, false
	}

	p.Logger.Printf(
		"Issued certificate for '%s', expires at %s, issued by '%s'",
		n.Unicode,
		c.Leaf.NotAfter.Format(time.RFC3339),
		c.Leaf.Issuer.CommonName,
	)

	return ProviderResult{
		Certificate: c,
	}, true
}

// IsValid returns true if the given provider result should still be considered
// valid.
//
// The behavior is undefined if the result was not obtained from this provider.
//
// This implementation considers the cached result invalid if the certificate is
// within 5 minutes of expiring. This is designed to mitigate the chance of
// serving a certificate that is close to expiry to a client that may consider
// the certificate to have already expired due to differences in system time.
func (p *AdhocProvider) IsValid(r ProviderResult) bool {
	t := time.Now().Add(-5 * time.Minute)
	return r.Certificate.Leaf.NotAfter.Before(t)
}
