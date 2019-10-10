package proxy

import (
	"net/http"
	"strings"

	"github.com/golang/gddo/httputil/header"
)

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
		"Upgrade",
		"Upgrade-Insecure-Requests":
		return true
	default:
		return false
	}
}
