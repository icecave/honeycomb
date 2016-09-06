package proxy

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/icecave/honeycomb/src/backend"
)

// NewWebSocketProxy creates a new proxy that forwards WebSocket requests.
func NewWebSocketProxy(logger *log.Logger) Proxy {
	return &webSocketProxy{
		websocket.DefaultDialer,
		&websocket.Upgrader{
			Error: func(
				writer http.ResponseWriter,
				_ *http.Request,
				statusCode int,
				_ error,
			) {
				WriteError(writer, statusCode)
			},
		},
		logger,
	}
}

type webSocketProxy struct {
	dialer   *websocket.Dialer
	upgrader *websocket.Upgrader
	logger   *log.Logger
}

// Serve forwards an HTTP request to a specific backend server.
func (proxy *webSocketProxy) ForwardRequest(
	endpoint *backend.Endpoint,
	frontendWriter *ResponseWriter,
	frontendRequest *http.Request,
) error {
	// Mangle the incoming request URL to point to the back-end ...
	url := *frontendRequest.URL
	url.Scheme = endpoint.GetScheme(true)
	url.Host = endpoint.Address

	// Connect to the back-end server ...
	backendConnection, backendResponse, err := proxy.dialer.Dial(
		url.String(),
		buildBackendHeaders(frontendRequest),
	)
	if err != nil {
		WriteError(frontendWriter, http.StatusBadGateway)
		return err
	}
	defer backendConnection.Close()

	// Strip out Hop-by-Hop headers from the backend's response to send to the
	// frontend connection ...
	upgradeHeaders := http.Header{}
	for name, values := range backendResponse.Header {
		if !isHopByHopHeader(name) {
			upgradeHeaders[name] = values
		}
	}

	// Upgrade the incoming connection to a websocket ...
	frontendWriter.StatusCode = http.StatusSwitchingProtocols
	frontendConnection, err := proxy.upgrader.Upgrade(
		frontendWriter,
		frontendRequest,
		upgradeHeaders,
	)
	if err != nil {
		return err
	}
	defer frontendConnection.Close()

	// Pipe data between the connections until they're closed ...
	return Pipe(
		backendConnection.UnderlyingConn(),
		frontendConnection.UnderlyingConn(),
	)
}
