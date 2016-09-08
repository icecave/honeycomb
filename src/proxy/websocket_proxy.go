package proxy

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/icecave/honeycomb/src/transaction"
)

// NewWebSocketProxy creates a new proxy that forwards WebSocket requests.
func NewWebSocketProxy() transaction.Handler {
	return &webSocketProxy{
		websocket.DefaultDialer,
		&websocket.Upgrader{
			Error: func(
				writer http.ResponseWriter,
				_ *http.Request,
				statusCode int,
				_ error,
			) {
				transaction.WriteStatusPage(writer, statusCode)
			},
		},
	}
}

type webSocketProxy struct {
	dialer   *websocket.Dialer
	upgrader *websocket.Upgrader
}

// Serve forwards an HTTP request to a specific backend server.
func (proxy *webSocketProxy) Serve(txn *transaction.Transaction) {
	// Mangle the incoming request URL to point to the back-end ...
	url := *txn.Request.URL
	url.Scheme = txn.Endpoint.GetScheme(true)
	url.Host = txn.Endpoint.Address

	// Connect to the back-end server ...
	backend, response, err := proxy.dialer.Dial(
		url.String(),
		buildBackendHeaders(txn.Request),
	)
	if err != nil {
		transaction.WriteStatusPage(txn.Writer, http.StatusBadGateway)
		txn.Error = err
		return
	}
	defer backend.Close()

	// Strip out Hop-by-Hop headers from the backend's response to send to the
	// frontend connection ...
	upgradeHeaders := http.Header{}
	for name, values := range response.Header {
		if !isHopByHopHeader(name) {
			upgradeHeaders[name] = values
		}
	}

	// Upgrade the incoming connection to a websocket ...
	client, err := proxy.upgrader.Upgrade(txn.Writer, txn.Request, upgradeHeaders)
	if err != nil {
		txn.Error = err
		return
	}
	defer client.Close()

	txn.HeadersSent(http.StatusSwitchingProtocols)

	// Pipe data between the connections until they're closed ...
	txn.Error = Pipe(
		client.UnderlyingConn(),
		backend.UnderlyingConn(),
	)
}
