package generator

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

// DefaultNotBeforeOffset is the default value used for a generator's
// NotBeforeOffset attribute when it is not specified. It is typically negative
// to allow for some clock-drift between client and server.
const DefaultNotBeforeOffset = -15 * time.Minute

// DefaultNotAfterOffset is the default value used for a generator's
// NotAfterOffset attribute when it is not specified.
const DefaultNotAfterOffset = 24 * time.Hour

// Generator creates new TLS certificates.
type Generator interface {
	// Generate creates a new TLS certificate for the given server name.
	Generate(context.Context, name.ServerName) (*tls.Certificate, error)
}
