package static

import (
	"context"

	"github.com/icecave/honeycomb/backend"
	"github.com/icecave/honeycomb/name"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type Locator []matcherEndpointPair

// Locate finds the back-end HTTP server for the given server name.
//
// It returns a score indicating the strength of the match. A value of 0 or
// less indicates that no match was made, in which case ep is nil.
//
// A non-zero score can be returned with a nil endpoint, indicating that the
// request should not be routed.
func (locator Locator) Locate(
	_ context.Context,
	serverName name.ServerName,
) (ep *backend.Endpoint, score int) {
	for _, item := range locator {
		if s := item.Matcher.Match(serverName); s > score {
			ep = item.Endpoint
			score = s
		}
	}

	return ep, score
}

// With returns a new StaticLocator that includes the given mapping.
func (locator Locator) With(pattern string, endpoint *backend.Endpoint) Locator {
	matcher, err := name.NewMatcher(pattern)
	if err != nil {
		panic(err)
	}

	return append(
		locator,
		matcherEndpointPair{matcher, endpoint},
	)
}

type matcherEndpointPair struct {
	Matcher  *name.Matcher
	Endpoint *backend.Endpoint
}
