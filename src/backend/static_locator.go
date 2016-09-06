package backend

import (
	"context"

	"github.com/icecave/honeycomb/src/name"
)

// StaticLocator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type StaticLocator []matcherEndpointPair

// Locate finds the back-end HTTP server for the given server name.
func (locator StaticLocator) Locate(_ context.Context, serverName name.ServerName) *Endpoint {
	for _, item := range locator {
		if item.Matcher.Match(serverName) {
			return item.Endpoint
		}
	}

	return nil
}

// With returns a new StaticLocator that includes the given mapping.
func (locator StaticLocator) With(pattern string, endpoint *Endpoint) StaticLocator {
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
	Endpoint *Endpoint
}
