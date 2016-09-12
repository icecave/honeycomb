package health

import (
	"io"
	"log"
	"net/http"

	"github.com/icecave/honeycomb/src/name"
)

const requestHost = "localhost"
const requestPath = "/.honeycomb/health-check"

// HTTPHandler is a http.Handler/frontend.ConditionalHandler that returns health
// check information.
type HTTPHandler struct {
	Checker Checker
	Logger  *log.Logger
}

// CanHandle returns true if request can be served by this handler.
func (handler *HTTPHandler) CanHandle(request *http.Request) bool {
	serverName, _ := name.FromHTTP(request)
	return serverName.Unicode == requestHost && request.URL.Path == requestPath
}

func (handler *HTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")

	status := Status{
		true,
		"The server is accepting requests, but no health-checker is configured.",
	}

	if handler.Checker != nil {
		status = handler.Checker.Check()
	}

	if status.IsHealthy {
		writer.WriteHeader(http.StatusOK)
	} else {
		if handler.Logger != nil {
			handler.Logger.Println(status)
		}

		writer.WriteHeader(http.StatusServiceUnavailable)
	}

	io.WriteString(writer, status.Message)
}
