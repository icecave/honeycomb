package proxy

import (
	"fmt"
	"io"
	"net/http"
)

// writeRequestHeaders writes the headers from request to writer.
func writeRequestHeaders(writer io.Writer, request *http.Request) error {
	if _, err := fmt.Fprintf(
		writer,
		"%s %s %s\r\n",
		request.Method,
		request.URL.RequestURI(),
		request.Proto,
	); err != nil {
		return err
	}

	if err := request.Header.Write(writer); err != nil {
		return err
	}

	_, err := io.WriteString(writer, "\r\n")
	return err
}

// writeResponseHeaders writes the headers from response to writer.
func writeResponseHeaders(
	writer http.ResponseWriter,
	response *http.Response,
	isWebSocket bool,
) {
	headers := writer.Header()
	for name, values := range response.Header {
		if !isHopByHopHeader(name) {
			headers[name] = values
		}
	}

	if isWebSocket {
		headers.Set("Connection", "upgrade")
		headers.Set("Upgrade", "websocket")
	}

	headers.Set("Strict-Transport-Security", "max-age=15768000")

	writer.WriteHeader(response.StatusCode)
}

// writeResponse writes the entirety of response to writer.
func writeResponse(writer http.ResponseWriter, response *http.Response) (int64, error) {
	defer response.Body.Close()
	writeResponseHeaders(writer, response, false)
	return io.Copy(writer, response.Body)
}
