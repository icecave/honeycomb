package static

import (
	"context"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type Locator []matcherEndpointPair

// Locate finds the back-end HTTP server for the given server name.
func (locator Locator) Locate(_ context.Context, serverName name.ServerName) *backend.Endpoint {
	for _, item := range locator {
		if item.Matcher.Match(serverName) {
			return item.Endpoint
		}
	}

	return nil
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
