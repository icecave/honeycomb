package generator

import (
	"context"
	"crypto/tls"
	"time"
)

// DefaultNotBeforeOffset is the default value used for a generator's
// NotBeforeOffset attribute when it is not specified. It is typically negative
// to allow for some clock-drift between client and server.
const DefaultNotBeforeOffset = -5 * time.Minute

// DefaultNotAfterOffset is the default value used for a generator's
// NotAfterOffset attribute when it is not specified.
//
// This is deliberately a short duration to ensure that generated certificates
// are discarded from the cache fairly often, giving the certificate resolver a
// chance to find a "real" certificate.
const DefaultNotAfterOffset = 10 * time.Minute

// Generator creates new TLS certificates.
type Generator interface {
	// Generate creates a new TLS certificate for the given server name.
	Generate(
		ctx context.Context,
		commonName string,
		dnsName string,
	) (*tls.Certificate, error)
}
