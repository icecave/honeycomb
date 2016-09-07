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

// Context stores the full context of the request across its lifetime.
type Context struct {
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
	// updates the context with information about the response.
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

	// LogError is an optional error that should be logged for this request.
	LogError error

	// Timer captures timing information of events during the request life-cycle.
	Timer Timer

	// BytesIn is the total number of bytes received for this request.
	// Includes websocket frames, but not HTTP headers.
	BytesIn int

	// BytesOut is the total number of bytes sent in response to this
	// request. Includes websocket frames, but not HTTP headers.
	BytesOut int
}

// NewContext creates a new context for the given request/response pair.
func NewContext(
	writer http.ResponseWriter,
	request *http.Request,
) *Context {
	ctx := &Context{
		Request:     request,
		IsWebSocket: websocket.IsWebSocketUpgrade(request),
		IsLogged:    true,
	}

	ctx.Timer.Start()

	ctx.Writer = &Writer{
		Inner:   writer,
		Context: ctx,
	}

	ctx.ServerName, ctx.LogError = name.FromHTTP(request)

	return ctx
}

// HeadersSent updates the context to reflect that the HTTP response headers
// have been sent.
func (ctx *Context) HeadersSent(statusCode int) {
	ctx.State = StateResponded
	ctx.StatusCode = statusCode
	ctx.Timer.FirstByteSent()
}

// Close marks the request as complete.
func (ctx *Context) Close() {
	ctx.State = StateClosed
	ctx.Timer.LastByteSent()
}

// String returns the log message for the context. If logging is disabled for
// this request, an empty string is returned. The log format consists of the
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
func (ctx *Context) String() string {
	var buffer bytes.Buffer

	// remote address
	write(buffer, ctx.Request.RemoteAddr)
	write(buffer, " ")

	// frontend
	if ctx.IsWebSocket {
		write(buffer, "wss://")
	} else {
		write(buffer, "https://")
	}
	write(buffer, " ")

	// backend + description
	if ctx.Endpoint == nil {
		write(buffer, "- - ")
	} else {
		write(buffer, ctx.Endpoint.GetScheme(ctx.IsWebSocket))
		write(buffer, "://")
		write(buffer, ctx.Endpoint.Address)
		write(buffer, " ")
		write(buffer, ctx.Endpoint.Description)
		write(buffer, " ")
	}

	// status code
	if ctx.StatusCode == 0 {
		write(buffer, "-")
	} else {
		write(buffer, strconv.Itoa(ctx.StatusCode))
	}

	// time to first / last byte
	switch ctx.State {
	case StateReceived:
		write(buffer, "- - ")
	case StateResponded:
		write(buffer, ctx.Timer.TimeToFirstByte.String())
		write(buffer, " ")
	case StateClosed:
		write(buffer, ctx.Timer.TimeToFirstByte.String())
		write(buffer, " ")
		write(buffer, ctx.Timer.TimeToLastByte.String())
		write(buffer, " ")
	}

	// bytes in / out
	write(buffer, strconv.Itoa(ctx.BytesIn))
	write(buffer, "i ")
	write(buffer, strconv.Itoa(ctx.BytesOut))
	write(buffer, "o ")

	// request information
	write(buffer, fmt.Sprintf(
		"%s %s %s",
		ctx.Request.Method,
		ctx.Request.URL,
		ctx.Request.Proto,
	))

	// error message (optional)
	if ctx.LogError != nil {
		write(buffer, " ")
		write(buffer, ctx.LogError.Error())
	} else if ctx.IsWebSocket && ctx.State == StateResponded {
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
