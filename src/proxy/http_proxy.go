package proxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/icecave/honeycomb/src/request"
)

// NewHTTPProxy creates a new proxy that forwards non-websocket requests to a
// back-end server.
func NewHTTPProxy(logger *log.Logger) request.Handler {
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
func (proxy *httpProxy) Serve(txn *request.Transaction) {
	// Clone the request with a new context to to store the honeycomb transaction.
	// This makes the transaction available to to HTTP round-tripper.
	req := txn.Request.WithContext(
		context.WithValue(
			txn.Request.Context(),
			"honeycomb",
			txn,
		),
	)

	// Copy the incoming request URL and mangle it to point to the endpoint.
	url := *req.URL
	req.URL = &url
	req.URL.Host = txn.Endpoint.Address
	req.URL.Scheme = txn.Endpoint.GetScheme(false)

	// @todo unify X-Forwarded-* headers with websocket proxy, this may not be
	// possible with httputil.ReverseProxy.
	proxy.reverseProxy.ServeHTTP(txn.Writer, req)
}

// RoundTrip is a custom implementation of http.RoundTripper (transport) that
// allow us to intercept the error from the transport before httputil.ReverseProxy
// gets a chance to send its response. This is done because the error page
// that ReverseProxy produces has no content.
func (proxy *httpProxy) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := proxy.transport.RoundTrip(req)
	if err != nil {
		txn := req.Context().Value("honeycomb").(*request.Transaction)
		txn.Error = err
		WriteError(txn.Writer, http.StatusBadGateway)
		txn.Close() // close the transaction early so reverse proxy can't send anything
	}

	return response, err
}
