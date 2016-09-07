package request

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
)

// Transaction stores the state of a HTTP request across its lifetime.
type Transaction struct {
	// ServerName is the SNI value provided during the TLS handshake.
	ServerName name.ServerName

	// Endpoint is the endpoint that the request should be routed to. If the
	// endpoint is unknown it will be nil.
	Endpoint *backend.Endpoint

	// State stores the current state of the request.
	State State

	// Request is the original HTTP request from the server.
	Request *http.Request

	// Writer is a wrapper around the the original HTTP response writer which
	// updates the transaction with information about the response.
	Writer *Writer

	// StatusCode is the HTTP status code sent in response to this request.
	// A value of zero means that no headers have been written or the request
	// has been hijacked.
	StatusCode int

	// IsWebSocket is true if the client provided HTTP headers that indicate a
	// websocket upgrade request.
	IsWebSocket bool

	// IsLogged indicates whether or not the result of the request should be
	// logged by the server.
	IsLogged bool

	// Error is the final error state of the request. If it is non-nil it is
	// logged.
	Error error

	// Timer captures timing information of events during the request life-cycle.
	Timer Timer

	// BytesIn is the total number of bytes received for this request.
	// Includes websocket frames, but not HTTP headers.
	BytesIn int

	// BytesOut is the total number of bytes sent in response to this
	// request. Includes websocket frames, but not HTTP headers.
	BytesOut int
}

// NewTransaction creates a new transaction for the given request/response pair.
func NewTransaction(
	writer http.ResponseWriter,
	request *http.Request,
) *Transaction {
	txn := &Transaction{
		Request:     request,
		IsWebSocket: websocket.IsWebSocketUpgrade(request),
		IsLogged:    true,
	}

	txn.Writer = &Writer{
		Inner:       writer,
		Transaction: txn,
	}

	txn.ServerName, txn.Error = name.FromHTTP(request)

	return txn
}

// Open starts the request.
func (txn *Transaction) Open() {
	txn.Timer.Start()
}

// HeadersSent updates the transaction to reflect that the HTTP response headers
// have been sent.
func (txn *Transaction) HeadersSent(statusCode int) {
	txn.Timer.FirstByteSent()
	txn.State = StateResponded
	txn.StatusCode = statusCode
}

// Close marks the request as complete. Call
func (txn *Transaction) Close() {
	if txn.State != StateClosed {
		txn.Timer.LastByteSent()
		txn.State = StateClosed
		txn.Writer.Inner = nil
	}
}

// String returns the log message for the transaction. If logging is disabled
// for this request, an empty string is returned. The log format consists of the
// following space separated fields:
//
// - remote address
// - frontend scheme + address
// - backend scheme + address
// - backend description
// - http status code
// - time to first byte
// - time to last byte
// - bytes inbound
// - bytes outbound
// - request information (method, URI and protocol)
// - message (optional)
//
// All fields are always present, except for the message. If a field value is
// unknown, a hyphen is used in place. If a string value itself contains spaces
// or double quotes it is represented as a double-quoted Go string. This allows
// log output to be parsed programatically.
func (txn *Transaction) String() string {
	var buffer bytes.Buffer

	// remote address
	write(buffer, txn.Request.RemoteAddr)
	write(buffer, " ")

	// frontend
	if txn.IsWebSocket {
		write(buffer, "wss://")
	} else {
		write(buffer, "https://")
	}
	write(buffer, " ")

	// backend + description
	if txn.Endpoint == nil {
		write(buffer, "- - ")
	} else {
		write(buffer, txn.Endpoint.GetScheme(txn.IsWebSocket))
		write(buffer, "://")
		write(buffer, txn.Endpoint.Address)
		write(buffer, " ")
		write(buffer, txn.Endpoint.Description)
		write(buffer, " ")
	}

	// status code
	if txn.StatusCode == 0 {
		write(buffer, "-")
	} else {
		write(buffer, strconv.Itoa(txn.StatusCode))
	}

	// time to first / last byte
	switch txn.State {
	case StateReceived:
		write(buffer, "- - ")
	case StateResponded:
		write(buffer, txn.Timer.TimeToFirstByte.String())
		write(buffer, " ")
	case StateClosed:
		write(buffer, txn.Timer.TimeToFirstByte.String())
		write(buffer, " ")
		write(buffer, txn.Timer.TimeToLastByte.String())
		write(buffer, " ")
	}

	// bytes in / out
	write(buffer, strconv.Itoa(txn.BytesIn))
	write(buffer, "i ")
	write(buffer, strconv.Itoa(txn.BytesOut))
	write(buffer, "o ")

	// request information
	write(buffer, fmt.Sprintf(
		"%s %s %s",
		txn.Request.Method,
		txn.Request.URL,
		txn.Request.Proto,
	))

	// error message (optional)
	if txn.Error != nil {
		write(buffer, " ")
		write(buffer, txn.Error.Error())
	} else if txn.IsWebSocket && txn.State == StateResponded {
		write(buffer, " websocket connection established")
	}

	return buffer.String()
}

// write is a helper function that writes to a string to a buffer, quoting the
// string if it contains whitespace or non-printable characters.
func write(buffer bytes.Buffer, str string) {
	quoted := strconv.Quote(str)
	if quoted != str || strings.ContainsRune(str, ' ') {
		buffer.WriteRune('"')
		buffer.WriteString(quoted)
		buffer.WriteRune('"')
	} else {
		buffer.WriteString(str)
	}
}
