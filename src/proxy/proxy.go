package proxy

import (
	"net/http"

	"github.com/icecave/honeycomb/src/transaction"
)

// Proxy forwards HTTP requests to back-end HTTP servers.
type Proxy struct {
	HTTPProxy      transaction.Handler
	WebSocketProxy transaction.Handler
}

// Serve dispatches the transaction to the appropriate proxy.
func (proxy *Proxy) Serve(txn *transaction.Transaction) {
	if txn.Endpoint == nil {
		WriteError(txn.Writer, http.StatusServiceUnavailable)
	} else if txn.IsWebSocket {
		proxy.WebSocketProxy.Serve(txn)
	} else {
		proxy.WebSocketProxy.Serve(txn)
	}
}
