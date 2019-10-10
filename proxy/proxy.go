package proxy

import "net/http"

// Proxy is a specialized HTTP handler that forwards to an"upstream" server.
type Proxy interface {
	// Forward proxies data between the client and the upstream server.
	Forward(
		writer http.ResponseWriter,
		request *http.Request,
		upstreamRequest *http.Request,
		log *LogContext,
	) error
}
