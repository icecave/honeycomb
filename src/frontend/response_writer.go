package frontend

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

// ResponseWriter wraps an http.ResponseWriter.
type ResponseWriter struct {
	Inner      http.ResponseWriter
	StatusCode int
	Size       int
	FirstWrite func(int)
}

// Header forwards to writer.Inner.Header()
func (writer *ResponseWriter) Header() http.Header {
	return writer.Inner.Header()
}

// Write forwards to writer.Inner.Write()
func (writer *ResponseWriter) Write(data []byte) (int, error) {
	if writer.StatusCode == 0 {
		writer.WriteHeader(http.StatusOK)
	}

	size, err := writer.Inner.Write(data)
	writer.Size += size

	return size, err
}

// WriteHeader forwards to writer.Inner.WriteHeader()
func (writer *ResponseWriter) WriteHeader(statusCode int) {
	writer.StatusCode = statusCode
	if writer.FirstWrite != nil {
		writer.FirstWrite(statusCode)
	}
	writer.Inner.WriteHeader(statusCode)
}

// Flush forwards to writer.Inner.Flush() if it implements http.Flusher(),
// otherwise it does nothing.
func (writer *ResponseWriter) Flush() {
	flusher, ok := writer.Inner.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

// Hijack fowards to writer.Inner.Hijack() if it implements http.Hijacker,
// otherwise it returns an error.
func (writer *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := writer.Inner.(http.Hijacker)
	if ok {
		return hijacker.Hijack()
	}

	return nil, nil, fmt.Errorf("The wrapped response does not implement http.Hijacker.")
}

// CloseNotify forwards to writer.Inner.CloseNotify(). A type assertion is
// performed on the writer.Inner to verify ti implements http.CloseNotifier.
func (writer *ResponseWriter) CloseNotify() <-chan bool {
	return writer.Inner.(http.CloseNotifier).CloseNotify()
}
