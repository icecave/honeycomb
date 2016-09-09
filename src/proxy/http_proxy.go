package proxy

import (
	"net/http"

	"github.com/icecave/honeycomb/src/statuspage"
)

// HTTPProxy is a proxy that handles regular non-websocket connections.
type HTTPProxy struct {
	Transport http.RoundTripper
}

// Forward proxies data between the client and the upstream server.
func (proxy *HTTPProxy) Forward(
	writer http.ResponseWriter,
	request *http.Request,
	upstreamRequest *http.Request,
	logContext *LogContext,
) error {
	transport := proxy.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	upstreamResponse, err := transport.RoundTrip(upstreamRequest)
	if err != nil {
		return statuspage.Error{Inner: err, StatusCode: http.StatusBadGateway}
	}

	logContext.Metrics.FirstByteSent()
	defer logContext.Metrics.LastByteSent()

	logContext.StatusCode = upstreamResponse.StatusCode
	logContext.Metrics.BytesIn = request.ContentLength // @todo handle -1 (content-length not known)
	logContext.Metrics.BytesOut, err = writeResponse(writer, upstreamResponse)

	return err
}
