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

	// Connect to the upstream server ...
	upstreamConnection, err := proxy.Dialer.Dial(upstreamRequest)
	if err != nil {
		return statuspage.Error{Inner: err, StatusCode: http.StatusBadGateway}
	}
	defer upstreamConnection.Close()

	// Re-add hop-by-hop headers that are needed for websockets ...
	upstreamRequest.Header.Set("Connection", "upgrade")
	upstreamRequest.Header.Set("Upgrade", "websocket")

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
	logContext.StatusCode = upstreamResponse.StatusCode

	// If the server is not switching protocols, proxy its response unchanged ...
	if upstreamResponse.StatusCode != http.StatusSwitchingProtocols {
		logContext.Metrics.BytesOut, err = writeResponse(writer, upstreamResponse)
		logContext.Metrics.LastByteSent()
		return err
	}

	// Otherwise return just the headers, then hijack the connection to proxy
	// the websocket frames ...
	writeResponseHeaders(writer, upstreamResponse, true)

	logContext.Log(nil)

	clientConnection, clientIO, err := hijacker.Hijack()
	if err != nil {
		return err
	}
	defer clientConnection.Close()

	return proxy.pipe(
		upstreamConnection,
		upstreamReader,
		clientConnection,
		clientIO.Reader,
		&logContext.Metrics,
	)
}

// pipe sends data between the upstream server and the client, first flushing
// any data that was buffered while reading the request and response headers.
func (proxy *WebSocketProxy) pipe(
	upstreamConnection io.ReadWriteCloser,
	upstreamReader *bufio.Reader,
	clientConnection io.ReadWriteCloser,
	clientReader *bufio.Reader,
	metrics *Metrics,
) error {
	done := make(chan error)
	go func() {
		bytes, err := proxy.copy(upstreamConnection, clientReader, clientConnection)
		metrics.BytesIn += bytes
		done <- err
	}()

	bytes, err := proxy.copy(clientConnection, upstreamReader, upstreamConnection)
	metrics.BytesOut += bytes
	metrics.LastByteSent()

	if e := <-done; e != nil {
		return e
	}

	return err
}

// copy first writes any buffered data from buffer to writer, then from reader
// until EOF is reached.
func (proxy *WebSocketProxy) copy(
	writer io.Writer,
	buffer *bufio.Reader,
	reader io.ReadCloser,
) (int64, error) {
	bufferedBytes := int64(buffer.Buffered())
	if bufferedBytes != 0 {
		if _, err := io.CopyN(writer, buffer, bufferedBytes); err != nil {
			return bufferedBytes, err
		}
	}

	bytes, err := io.Copy(writer, reader)
	reader.Close()

	return bufferedBytes + bytes, err
}
