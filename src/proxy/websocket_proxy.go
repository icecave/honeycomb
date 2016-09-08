package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/icecave/honeycomb/src/transaction"
)

// WebSocketProxy is a transaction.Handler that proxies websocket requests.
type WebSocketProxy struct{}

// Serve forwards an HTTP request to a specific backend server.
func (proxy *WebSocketProxy) Serve(txn *transaction.Transaction) {
	// Attempt to establish a connection to the server ...
	backend, err := txn.Endpoint.Dial(txn.Request.Context())
	if err != nil {
		txn.Error = err
		return
	}
	defer backend.Close()

	// Forward the request headers from the client to the backend ...
	err = proxy.sendRequestHeaders(txn, backend)
	if err != nil {
		txn.Error = err
		return
	}

	// Read the response from the backend ...
	reader := bufio.NewReader(backend)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		txn.Error = err
		return
	}

	// Forward the response headers from the backend to the client ...
	proxy.sendResponseHeaders(txn, response)

	// If we're not switching to a websocket connection, forward the entire
	// body, then return ...
	if response.StatusCode != http.StatusSwitchingProtocols {
		_, txn.Error = io.Copy(txn.Writer, response.Body)
		return
	}

	// Otherwise, hijack the client's connection to create a bidirectional pipe
	// between the client and the backend ...
	client, _, err := txn.Writer.Hijack()
	if err != nil {
		txn.Error = err
		return
	}
	defer client.Close()

	// Some frame data may have already been read from the backend while reading
	// the headers, flush any such data first ...
	if n := reader.Buffered(); n > 0 {
		_, err := io.CopyN(client, reader, int64(n))
		if err != nil {
			txn.Error = err
			return
		}
	}

	// Pipe the data ...
	txn.Error = Pipe(backend, client)
}

func (proxy *WebSocketProxy) sendRequestHeaders(
	txn *transaction.Transaction,
	backend net.Conn,
) error {
	_, err := fmt.Fprintf(
		backend,
		"GET %s HTTP/1.1\r\n",
		txn.Request.URL.RequestURI(),
	)
	if err != nil {
		return err
	}

	headers := prepareHeaders(txn)
	headers.Set("Connection", "Upgrade")
	headers.Set("Upgrade", "websocket")

	err = headers.Write(backend)
	if err != nil {
		return err
	}

	_, err = io.WriteString(backend, "\r\n")
	return err
}

func (proxy *WebSocketProxy) sendResponseHeaders(
	txn *transaction.Transaction,
	response *http.Response,
) {
	headers := txn.Writer.Header()

	for name, value := range response.Header {
		headers[name] = value
	}

	txn.Writer.WriteHeader(response.StatusCode)
}
