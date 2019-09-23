package cert

import (
	"crypto/tls"
	"fmt"
)

// AggregateProvider is a collection providers that are tried in order.
type AggregateProvider []Provider

// GetCertificate attempts to fetch a certificate for the given request.
func (p AggregateProvider) GetCertificate(
	info *tls.ClientHelloInfo,
) (*tls.Certificate, error) {
	for _, pr := range p {
		c, err := pr.GetCertificate(info)
		if err == nil {
			return c, err
		}
	}

	return nil, fmt.Errorf(
		"no certificate provider was able to provide a certificate for %s",
		info.ServerName,
	)
}
