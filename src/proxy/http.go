package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/icecave/honeycomb/src/backend"
)

// NewHTTPProxy creates a new proxy that forwards non-websocket requests to a
// back-end server.
func NewHTTPProxy(logger *log.Logger) Proxy {
	return &httpProxy{
		reverseProxy: httputil.ReverseProxy{
			Director: func(*http.Request) {},
			ErrorLog: logger,
		},
	}
}

type httpProxy struct {
	reverseProxy httputil.ReverseProxy
}

// Serve forwards an HTTP request to a specific back-end server.
func (proxy *httpProxy) ForwardRequest(
	endpoint *backend.Endpoint,
	response http.ResponseWriter,
	request *http.Request,
) error {
	// Mangle the incoming request URL to point to the back-end ...
	request.URL.Host = endpoint.Address
	request.URL.Scheme = endpoint.GetScheme(false)

	// @todo use buildBackendHeaders for consistency.
	proxy.reverseProxy.ServeHTTP(response, request)

	// @todo map 502 bad gateway produced to an error that can be handled
	// by the layer above.
	return nil
}
