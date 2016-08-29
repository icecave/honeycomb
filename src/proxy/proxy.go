package proxy

import (
	"net/http"

	"github.com/icecave/honeycomb/src/backend"
)

// Proxy forwards HTTP requests to back-end HTTP servers.
type Proxy interface {
	// ForwardRequest forwards an HTTP request to a specific back-end server.
	ForwardRequest(
		endpoint *backend.Endpoint,
		response http.ResponseWriter,
		request *http.Request,
	) error
}
