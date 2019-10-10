package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// WebSocketDialer connects to an upstream websocket server.
type WebSocketDialer interface {
	// Dial connects to the websocket server described by request.
	Dial(*http.Request) (net.Conn, error)
}

// BasicWebSocketDialer is the default WebSocketDialer implementation.
type BasicWebSocketDialer struct {
	Dialer    *net.Dialer
	TLSConfig *tls.Config
}

// Dial connects to the websocket server described by request.
func (dialer *BasicWebSocketDialer) Dial(
	request *http.Request,
) (net.Conn, error) {
	actual := dialer.Dialer
	if actual == nil {
		actual = &net.Dialer{}
	}

	if strings.EqualFold(request.URL.Scheme, "wss") {
		if deadline, ok := request.Context().Deadline(); ok {
			actual.Deadline = deadline
		}

		return tls.DialWithDialer(
			actual,
			"tcp",
			request.URL.Host,
			dialer.TLSConfig,
		)
	}

	return actual.DialContext(
		request.Context(),
		"tcp",
		request.URL.Host,
	)
}
