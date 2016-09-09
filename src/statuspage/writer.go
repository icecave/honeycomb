package statuspage

import "net/http"

// DefaultWriter is the status page writer that is used if no other is specified.
var DefaultWriter Writer = &TemplateWriter{}

// Writer writes HTTP status pages to an HTTP response writer.
type Writer interface {
	// Write outputs an HTTP status page for statusCode to writer, in response
	// to request.
	Write(
		writer http.ResponseWriter,
		request *http.Request,
		statusCode int,
	) (bodySize int64, err error)

	// WriteMessage outputs an HTTP status page for statusCode to writer, in
	// response to request, including a custom message.
	WriteMessage(
		writer http.ResponseWriter,
		request *http.Request,
		statusCode int,
		message string,
	) (bodySize int64, err error)

	// WriteError outputs an appropriate HTTP status page for the given error to
	// writer, in response to request.
	WriteError(
		writer http.ResponseWriter,
		request *http.Request,
		statusErr error,
	) (statusCode int, bodySize int64, err error)
}
