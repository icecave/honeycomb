package proxy

import (
	"net"
	"net/http"
	"strings"

	"github.com/golang/gddo/httputil/header"
)

// prepareUpstreamHeaders produces a copy of request.Header and modifies them so that
// they are suitable to send to the upstream server.
func prepareUpstreamHeaders(request *http.Request) (http.Header, bool) {
	upstreamHeaders := http.Header{}
	forwardedFor, _, _ := net.SplitHostPort(request.RemoteAddr)

	for name, values := range request.Header {
		if name == "X-Forwarded-For" {
			forwardedFor = strings.Join(values, ", ") + ", " + forwardedFor
		} else if !isHopByHopHeader(name) {
			upstreamHeaders[name] = values
		}
	}

	upstreamHeaders.Set("X-Forwarded-For", forwardedFor)
	upstreamHeaders.Set("Host", request.Host)

	isWebSocketRequest := isWebSocketUpgrade(request.Header)
	if isWebSocketRequest {
		upstreamHeaders.Set("Connection", "upgrade")
		upstreamHeaders.Set("Upgrade", "websocket")
	}

	return upstreamHeaders, isWebSocketRequest
}

// isWebSocketUpgrade checks whether the given HTTP headers indicate a websocket
// upgrade request or response.
func isWebSocketUpgrade(headers http.Header) bool {
	isUpgrade := false
	for _, value := range header.ParseList(headers, "Connection") {
		if strings.EqualFold(value, "upgrade") {
			isUpgrade = true
			break
		}
	}

	if isUpgrade {
		for _, value := range header.ParseList(headers, "Upgrade") {
			if strings.EqualFold(value, "websocket") {
				return true
			}
		}
	}

	return false
}

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
