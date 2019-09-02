package backend

import (
	"context"

	"github.com/icecave/honeycomb/src/name"
)

// AggregateLocator combines multiple locators to find endpoints.
type AggregateLocator []Locator

// Locate finds the back-end HTTP server for the given server name.
//
// It returns a score indicating the strength of the match. A value of 0 or
// less indicates that no match was made, in which case ep is nil.
//
// A non-zero score can be returned with a nil endpoint, indicating that the
// request should not be routed.
func (locator AggregateLocator) Locate(
	ctx context.Context,
	serverName name.ServerName,
) (ep *Endpoint, score int) {
	for _, loc := range locator {
		if e, s := loc.Locate(ctx, serverName); s > score {
			ep = e
			score = s
		}
	}

	return ep, score
}
