package proxy

import (
	"net"
	"net/http"
	"strings"
)

// isHopByHopHeader checks if a given header name is a Hop-by-Hop header, and
// hence should not be forwarded to back-end servers. The name must already be
// canonicalized with http.CanonicalHeaderKey().
func isHopByHopHeader(name string) bool {
	switch name {
	case "Sec-Websocket-Key",
		"Sec-Websocket-Version",
		"Sec-Websocket-Accept",

		// The remaininder header list was lifted from httputil.ReverseProxy
		// https://golang.org/src/net/http/httputil/reverseproxy.go
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

// buildBackendHeaders creates a set of headers that are to be forwarded to the
// backend server for the given request. The X-Forwarded-For header is added.
func buildBackendHeaders(request *http.Request) http.Header {
	// @todo add X-Forwarded-Port, X-Forwarded-Proto
	headers := http.Header{}
	hasXForwardedFor := false
	clientIP, _, _ := net.SplitHostPort(request.RemoteAddr)

	for name, values := range request.Header {
		if !isHopByHopHeader(name) {
			headers[name] = values
		} else if !hasXForwardedFor && name == "X-Forwarded-For" {
			chain := strings.Join(values, ", ") + ", " + clientIP
			headers.Set(name, chain)
			hasXForwardedFor = true
		}
	}

	if !hasXForwardedFor {
		headers.Set("X-Forwarded-For", clientIP)
	}

	return headers
}
