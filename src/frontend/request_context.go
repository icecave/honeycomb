package frontend

import (
	"net/http"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/proxy"
)

type requestContext struct {
	Writer      proxy.ResponseWriter
	Request     *http.Request
	IsWebSocket bool
	Timer       requestTimer
	Endpoint    *backend.Endpoint
	Error       error
}
