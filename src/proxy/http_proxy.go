package proxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/icecave/honeycomb/src/backend"
)

// NewHTTPProxy creates a new proxy that forwards non-websocket requests to a
// back-end server.
func NewHTTPProxy(logger *log.Logger) Proxy {
	proxy := &httpProxy{transport: http.DefaultTransport}
	proxy.reverseProxy = httputil.ReverseProxy{
		Director:  func(*http.Request) {},
		ErrorLog:  logger,
		Transport: proxy,
	}

	return proxy
}

type httpProxy struct {
	reverseProxy httputil.ReverseProxy
	transport    http.RoundTripper
}

// Serve forwards an HTTP request to a specific back-end server.
func (proxy *httpProxy) ForwardRequest(
	endpoint *backend.Endpoint,
	writer *ResponseWriter,
	request *http.Request,
) error {
	// Mangle the incoming request URL to point to the back-end ...
	request.URL.Host = endpoint.Address
	request.URL.Scheme = endpoint.GetScheme(false)

	// Store the writer on the request so we can use it in RoundTrip() ...
	request = request.WithContext(
		context.WithValue(
			request.Context(),
			"writer",
			writer,
		),
	)

	// @todo unify X-Forwarded-* headers with websocket proxy, this may not be
	// possible with httputil.ReverseProxy.
	proxy.reverseProxy.ServeHTTP(writer, request)

	return nil
}

// RoundTrip is a custom implementation of http.RoundTripper (transport) that
// allow us to intercept the error from the transport before httputil.ReverseProxy
// gets a chance to send its response. This is done because the error page
// that ReverseProxy produces has no content.
func (proxy *httpProxy) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := proxy.transport.RoundTrip(request)
	if err != nil {
		writer := request.Context().Value("writer").(*ResponseWriter)
		WriteError(writer, http.StatusBadGateway)
		writer.Inner = nil
	}

	return response, err
}
