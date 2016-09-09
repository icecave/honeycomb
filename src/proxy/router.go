package proxy

import (
	"net/http"
	"net/url"
)

// Router is an interface for determine which upstream server to connect to.
type Router interface {
	// Route updates upstreamURL and upstreamHeaders as appropriate for the
	// upstream server, based on request.
	Route(
		request *http.Request,
		isWebSocketRequest bool,
		upstreamURL *url.URL,
		upstreamHeaders http.Header,
	) (info string, err error)
}
