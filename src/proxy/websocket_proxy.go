package proxy

import (
	"bufio"
	"errors"
	"io"
	"net/http"

	"github.com/icecave/honeycomb/src/statuspage"
)

// WebSocketProxy is a proxy that handles WebSocket connections.
type WebSocketProxy struct {
	Dialer WebSocketDialer
}

// Forward proxies data between the client and the upstream server.
func (proxy *WebSocketProxy) Forward(
	writer http.ResponseWriter,
	request *http.Request,
	upstreamRequest *http.Request,
	logContext *LogContext,
) error {
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		return errors.New("client connection can not be hijacked")
	}

	// Connect to theupstream server ...
	dialer := proxy.Dialer
	if dialer == nil {
		dialer = DefaultWebSocketDialer
	}
	upstreamConnection, err := dialer.Dial(upstreamRequest)
	if err != nil {
		return statuspage.Error{Inner: err, StatusCode: http.StatusBadGateway}
	}
	defer upstreamConnection.Close()

	// Send the HTTP request ...
	err = writeRequestHeaders(upstreamConnection, upstreamRequest)
	if err != nil {
		return statuspage.Error{Inner: err, StatusCode: http.StatusBadGateway}
	}

	// Read the server's http response ...
	upstreamReader := bufio.NewReader(upstreamConnection)
	upstreamResponse, err := http.ReadResponse(upstreamReader, upstreamRequest)
	if err != nil {
		return statuspage.Error{Inner: err, StatusCode: http.StatusBadGateway}
	}

	logContext.Metrics.FirstByteSent()
	defer logContext.Metrics.LastByteSent()

	logContext.StatusCode = upstreamResponse.StatusCode

	// If the server is not switching protocols, proxy its response unchanged ...
	if upstreamResponse.StatusCode != http.StatusSwitchingProtocols {
		logContext.Metrics.BytesOut, err = writeResponse(writer, upstreamResponse)
		return err
	}

	// Otherwise return just the headers, then hijack the connection to proxy
	// the websocket frames ...
	writeResponseHeaders(writer, upstreamResponse)

	logContext.Log(nil)

	// @todo read buffered input from client connection before hijacking
	clientConnection, _, err := hijacker.Hijack()
	if err != nil {
		return err
	}
	defer clientConnection.Close()

	// Some frame data may have already been read from the upstream server while
	// reading the response headers, flush any such data first ...
	if n := upstreamReader.Buffered(); n > 0 {
		logContext.Metrics.BytesOut, err = io.CopyN(clientConnection, upstreamReader, int64(n))
		if err != nil {
			return err
		}
	}

	bytesIn, bytesOut, err := pipe(upstreamConnection, clientConnection)
	logContext.Metrics.BytesIn += bytesIn
	logContext.Metrics.BytesOut += bytesOut

	return err
}
