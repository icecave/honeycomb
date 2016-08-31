package proxy

import (
	"io"
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
	writer *ResponseWriter,
	request *http.Request,
) error {
	// Mangle the incoming request URL to point to the back-end ...
	request.URL.Host = endpoint.Address
	request.URL.Scheme = endpoint.GetScheme(false)

	// @todo unify X-Forwarded-* headers with websocket proxy, this may not be
	// possible with httputil.ReverseProxy.
	proxy.reverseProxy.ServeHTTP(writer, request)

	// httputil.ReverseProxy writes a bad gateway response with no body. We
	// can't change the code, so at least add a response body. This should
	// probably be a 504 Gateway Timeout in some circumstances anyway.
	// @todo get access to the internal error to send the appropriate code and/or
	// get rid of httputil.ReverseProxy entirely :/
	if writer.StatusCode == http.StatusBadGateway && writer.Size == 0 {
		io.WriteString(writer, "Bad Gateway")
	}

	return nil
}
