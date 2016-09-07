package proxy

import (
	"net/http"

	"github.com/icecave/honeycomb/src/request"
)

// Proxy forwards HTTP requests to back-end HTTP servers.
type Proxy struct {
	HTTPProxy      request.Handler
	WebSocketProxy request.Handler
}

// Serve dispatches the transaction to the appropriate proxy.
func (proxy *Proxy) Serve(txn *request.Transaction) {
	if txn.Endpoint == nil {
		WriteError(txn.Writer, http.StatusServiceUnavailable)
	} else if txn.IsWebSocket {
		proxy.WebSocketProxy.Serve(txn)
	} else {
		proxy.WebSocketProxy.Serve(txn)
	}
}
