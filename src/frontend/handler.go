package frontend

import (
	"log"
	"net/http"

	"github.com/icecave/honeycomb/src/statuspage"
)

// Handler provides the main http.Handler implementation.
type Handler struct {
	Proxy            http.Handler
	HealthCheck      ConditionalHandler
	StatusPageWriter statuspage.Writer
	Logger           *log.Logger
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if handler.HealthCheck != nil && handler.HealthCheck.CanHandle(request) {
		handler.HealthCheck.ServeHTTP(writer, request)
	} else {
		handler.Proxy.ServeHTTP(writer, request)
	}
}
