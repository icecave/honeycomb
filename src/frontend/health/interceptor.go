package health

import (
	"io"

	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/transaction"
)

const healthCheckPath = "/health"

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

// Intercept may optionally handle the request. The interceptor may also
// clear txn.Error if the error no longer applies once the request is
// intercepted.
func (in *Interceptor) Intercept(txn *transaction.Transaction) {
	if !in.Provides(txn.ServerName) {
		return
	} else if txn.Request.URL.Path != healthCheckPath {
		return
	}

	txn.IsLogged = false
	txn.Error = nil

	// The fact that this request made it through is enough to verify that the
	// HTTP server is listening ...
	txn.Writer.Header().Add("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(txn.Writer, "Server is accepting requests.")
	txn.Close()

	// @todo add some basic stats
	// @todo check that we're connected to a docker swarm master
}
