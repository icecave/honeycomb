package frontend

import (
	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/request"
)

// Interceptor is an http.Handler that conditionally intercepts HTTPS requests
// before they are routed to a back-end.
type Interceptor interface {
	// Provides checks if this interceptor provides services for the given
	// server name. This allows the interceptor to service requests for server
	// names that are not routed to any endpoints. An interceptor need not
	// "provide" a server name in order to intercept its request
	Provides(name.ServerName) bool

	// Intercept may optionally handle the request. The interceptor may also
	// clear txn.Error if the error no longer applies once the request is
	// intercepted.
	Intercept(*request.Transaction)
}
