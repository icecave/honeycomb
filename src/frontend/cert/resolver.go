package cert

import (
	"crypto/tls"

	"github.com/icecave/honeycomb/src/name"
)

// Resolver uses one or more certificate providers to obtain certificates for
// TLS/HTTPS requests.
//
// The GetCertificate() method matches the signature of the
// tls.Config.GetCertificate hook.
type Resolver struct {
	// IsRecognized is a hook that returns true if a server name is recognized
	// by the proxy, that is, there is some backend service to which the server
	// name is routed. If IsRecognized is nil, all server names are treated as
	// unrecognized.
	IsRecognized func(name.ServerName) bool

	// Recognized is an ordered list of providers used to obtain certificates
	// for recognized server names.
	Recognized []Provider

	// Unrecognized is an ordered list of providers used to obtain certificates
	// for unrecognized server names.
	Unrecognized []Provider

	cache Cache
}

// GetCertificate returns the certificate used for a TLS request.
func (r *Resolver) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	n, err := name.TryParse(info.ServerName)
	if err != nil {
		return nil, err
	}

	isRecognized := r.isRecognized(n)
	worstRank := len(r.Recognized)

	// If the server name is unrecognized we will accept "worse" entries.
	if !isRecognized {
		worstRank += len(r.Unrecognized)
	}

	// If there's a valid entry in the cache, use it.
	if pr, ok := r.cache.Get(n, worstRank); ok {
		return pr.Certificate, nil
	}

	// Otherwise, we will try to obtain a certificate.
	providers := r.Recognized
	rankOffset := 0
	if !isRecognized {
		providers = r.Unrecognized
		rankOffset += len(r.Recognized)
	}

	for rank, p := range providers {
		pr, ok := p.GetCertificate(n, info)

		if ok {
			if !pr.ExcludeFromCache {
				r.cache.Put(
					n,
					rankOffset+rank,
					p,
					pr,
				)
			}

			return pr.Certificate, nil
		}
	}

	// There was no error, as such, but we were unable to provide a certificate.
	// Go's TLS implementation will fall-back to other certificate search
	// methods, eventually landing on the default certificate.
	return nil, nil
}

// isRecognized returns true if n is a recognized server name.
func (r *Resolver) isRecognized(n name.ServerName) bool {
	if r.IsRecognized == nil {
		return false
	}

	return r.IsRecognized(n)
}
