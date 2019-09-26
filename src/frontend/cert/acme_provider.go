package cert

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/icecave/honeycomb/src/name"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// ACMEProvider is an implementation of Provider that obtains certificates from
// an ACME provider, such as Let's Encrypt.
type ACMEProvider struct {
	// Manager is the ACME certificate manager.
	Manager *autocert.Manager

	// Logger is the destination for messages about the acquired certificates,
	// and any errors that occur when acquiring certificates.
	Logger *log.Logger
}

// GetCertificate obtains a certificate from an ACME server.
func (p *ACMEProvider) GetCertificate(
	n name.ServerName,
	info *tls.ClientHelloInfo,
) (ProviderResult, bool) {
	c, err := p.Manager.GetCertificate(info)
	if err != nil {
		p.Logger.Printf(
			"Unable to acquire certificate for '%s' via ACME: %s",
			n.Unicode,
			err,
		)

		return ProviderResult{}, false
	}

	p.Logger.Printf(
		"Acquired certificate for '%s' via ACME, expires at %s, issued by '%s'",
		n.Unicode,
		c.Leaf.NotAfter.Format(time.RFC3339),
		c.Leaf.Issuer.CommonName,
	)

	return ProviderResult{
		Certificate: c,
		ExcludeFromCache: len(info.SupportedProtos) == 1 &&
			info.SupportedProtos[0] == acme.ALPNProto,
	}, true
}

// IsValid returns true if the given provider result should still be considered
// valid.
//
// The behavior is undefined if the result was not obtained from this provider.
//
// This implementation considers the cached result invalid if it's within
// p.Manager.RenewBefore of expiry. As per the autocert.Manager documentation,
// if RenewBefore is zero, a default of 30 days is used.
func (p *ACMEProvider) IsValid(r ProviderResult) bool {
	d := p.Manager.RenewBefore
	if d == 0 {
		d = 30 * 24 * time.Hour
	}

	t := time.Now().Add(-d)

	return r.Certificate.Leaf.NotAfter.Before(t)
}
