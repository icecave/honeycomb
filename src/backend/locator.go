package backend

import (
	"context"

	"github.com/icecave/honeycomb/src/name"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type Locator interface {
	// Locate finds the back-end HTTP server for the given server name.
	Locate(ctx context.Context, serverName name.ServerName) *Endpoint
}
