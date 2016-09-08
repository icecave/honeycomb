package proxy

import (
	"net"
	"net/http"
	"strings"

	"github.com/icecave/honeycomb/src/transaction"
)

// isHopByHopHeader checks if a given header name is a Hop-by-Hop header, and
// hence should not be forwarded to back-end servers. The name must already be
// canonicalized with http.CanonicalHeaderKey().
func isHopByHopHeader(name string) bool {
	switch name {
	case
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade":
		return true
	default:
		return false
	}
}

// prepareHeaders creates a set of headers that are to be forwarded to the
// backend server for the given transaction. The X-Forwarded-For header is added.
func prepareHeaders(txn *transaction.Transaction) http.Header {
	headers := http.Header{}
	forwardedFor, _, _ := net.SplitHostPort(txn.Request.RemoteAddr)

	for name, values := range txn.Request.Header {
		if name == "X-Forwarded-For" {
			forwardedFor = strings.Join(values, ", ") + ", " + forwardedFor
		} else if !isHopByHopHeader(name) {
			headers[name] = values
		}
	}

	headers["X-Forwarded-For"] = []string{forwardedFor}
	headers["Host"] = []string{txn.Endpoint.Address}

	return headers
}
