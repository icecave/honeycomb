package proxy

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/icecave/honeycomb/src/transaction"
)

// NewHTTPProxy creates a new proxy that forwards non-websocket requests to a
// back-end server.
func NewHTTPProxy() transaction.Handler {
	proxy := &httpProxy{transport: http.DefaultTransport}
	proxy.reverseProxy = httputil.ReverseProxy{
		Director:  func(*http.Request) {},
		ErrorLog:  log.New(ioutil.Discard, "", 0),
		Transport: proxy,
	}

	return proxy
}

type httpProxy struct {
	reverseProxy httputil.ReverseProxy
	transport    http.RoundTripper
}

// Serve forwards an HTTP request to a specific back-end server.
func (proxy *httpProxy) Serve(txn *transaction.Transaction) {
	// Clone the request with a new context to to store the honeycomb transaction.
	// This makes the transaction available to to HTTP round-tripper.
	request := txn.Request.WithContext(
		context.WithValue(
			txn.Request.Context(),
			"txn",
			txn,
		),
	)

	// Copy the incoming request URL and mangle it to point to the endpoint.
	url := *request.URL
	request.URL = &url
	request.URL.Host = txn.Endpoint.Address
	request.URL.Scheme = txn.Endpoint.GetScheme(false)

	// @todo unify X-Forwarded-* headers with websocket proxy, this may not be
	// possible with httputil.ReverseProxy.
	proxy.reverseProxy.ServeHTTP(txn.Writer, request)
}

// RoundTrip is a custom implementation of http.RoundTripper (transport) that
// allow us to intercept the error from the transport before httputil.ReverseProxy
// gets a chance to send its response. This is done because the error page
// that ReverseProxy produces has no content.
func (proxy *httpProxy) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := proxy.transport.RoundTrip(request)
	if err != nil {
		txn := request.Context().Value("txn").(*transaction.Transaction)
		txn.Error = err
		transaction.WriteStatusPage(txn.Writer, http.StatusBadGateway)
		txn.Close() // close the transaction early so reverse proxy can't send anything
	}

	return response, err
}
