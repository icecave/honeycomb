package backend

import (
	"context"

	"github.com/icecave/honeycomb/name"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type Locator interface {
	// Locate finds the back-end HTTP server for the given server name.
	//
	// It returns a score indicating the strength of the match. A value of 0 or
	// less indicates that no match was made, in which case ep is nil.
	//
	// A non-zero score can be returned with a nil endpoint, indicating that the
	// request should not be routed.
	Locate(ctx context.Context, serverName name.ServerName) (ep *Endpoint, score int)
}
