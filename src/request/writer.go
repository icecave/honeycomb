package request

import (
	"bufio"
	"fmt"

	"net"
	"net/http"
)

// Writer is an http.ResponseWriter that wraps another response writer, trapping
// information about the response before it is written.
//
// The interface supports methods from http.ResponseWriter, http.Flusher,
// http.Hijacker and http.CloseNotifier.
type Writer struct {
	// Inner is the original response writer to which the response is written.
	// If Inner is nil, the response is considered closed and no additional data
	// is sent to the client.
	Inner http.ResponseWriter

	// Transaction is request transaction to which this writer belongs.
	Transaction *Transaction
}

// Header returns writer.Inner.Header; or nil if the writer is closed.
func (writer *Writer) Header() http.Header {
	if writer.Inner != nil {
		return writer.Inner.Header()
	}

	return nil
}

// Write sends data to the client; unless the writer is closed. If data is
// written before the HTTP headers have been sent, a response code of 200 OK is
// used.
func (writer *Writer) Write(data []byte) (int, error) {
	if writer.Inner == nil {
		return 0, nil
	}

	if writer.Transaction.State == StateReceived {
		writer.WriteHeader(http.StatusOK)
	}

	size, err := writer.Inner.Write(data)
	writer.Transaction.BytesOut += size

	return size, err
}

// WriteHeader sends the HTTP headers, unless the response is closed. If an
// OnResponse handler is set, it is called with the status code.
func (writer *Writer) WriteHeader(statusCode int) {
	if writer.Inner != nil {
		writer.Inner.WriteHeader(statusCode)
		writer.Transaction.HeadersSent(statusCode)
	}
}

// Flush calls writer.Inner.Flush() if it implements http.Flusher and the writer
// is not closed.
func (writer *Writer) Flush() {
	if writer.Inner != nil {
		if flusher, ok := writer.Inner.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// Hijack calls writer.Inner.Hijack() if it implements http.Hijacker and
// the response is not closed, otherwise it returns an error.
func (writer *Writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if writer.Inner == nil {
		return nil, nil, fmt.Errorf("The response writer is closed.")
	}

	if hijacker, ok := writer.Inner.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, fmt.Errorf("The inner response writer does not implement http.Hijacker.")
}

// CloseNotify attempts to cast writer.Inner to a http.CloseNotifier and call
// the CloseNotify() method.
func (writer *Writer) CloseNotify() <-chan bool {
	return writer.Inner.(http.CloseNotifier).CloseNotify()
}
