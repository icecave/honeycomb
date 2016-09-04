package health

import (
	"io"

	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/name"
)

// Interceptor intercepts incoming requests to https://localhost/health and
// returns a basic health status of the server, suitable for use with Docker
// health checks.
type Interceptor struct{}

// Provides checks if this interceptor provides services for the given
// server name. This allows the interceptor to service requests for server
// names that are not routed to any endpoints. An interceptor need not
// "provide" a server name in order to intercept its request
func (in *Interceptor) Provides(serverName name.ServerName) bool {
	return serverName.Unicode == "localhost"
}

// Intercept may optionally handle the request. If the request is handled,
// ctx.Intercept must be set to true. The interceptor may also clear
// ctx.Error if the error no longer applies once the request is intercepted.
func (in *Interceptor) Intercept(ctx *frontend.RequestContext) {
	if !in.Provides(ctx.ServerName) {
		return
	} else if ctx.Request.URL.Path != "/health" {
		return
	}

	ctx.Intercepted = true
	ctx.Error = nil

	// The fact that this request made it through is enough to verify that the
	// HTTP server is listening ...
	ctx.Writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(&ctx.Writer, "Server is accepting requests.") // @todo add some basic stats
}
