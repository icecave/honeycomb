package proxy

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/statuspage"
)

// Handler is an http.Handler that proxies requests to an upstream server.
type Handler struct {
	Locator                backend.Locator
	SecureHTTPProxy        Proxy
	InsecureHTTPProxy      Proxy
	SecureWebSocketProxy   Proxy
	InsecureWebSocketProxy Proxy
	StatusPageWriter       statuspage.Writer
	Logger                 *log.Logger
}

// ServeHTTP proxies the request to the appropriate upstream server.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logContext := &LogContext{Logger: handler.Logger, Request: request}
	logContext.Metrics.Start()

	err := handler.forward(writer, request, logContext)

	// If there was an error and no response has been sent, send an error page.
	if err != nil && logContext.StatusCode == 0 {
		handler.writeStatusPage(writer, request, logContext, err)
	}

	logContext.Log(err)
}

func (handler *Handler) forward(
	writer http.ResponseWriter,
	request *http.Request,
	logContext *LogContext,
) (err error) {
	isWebSocket := isWebSocketUpgrade(request.Header)
	logContext.IsWebSocket = isWebSocket

	endpoint, err := handler.locate(request)
	if err != nil {
		return
	}

	logContext.Endpoint = endpoint

	proxy := handler.selectProxy(endpoint, isWebSocket)

	return proxy.Forward(
		writer,
		request,
		handler.prepareUpstreamRequest(request, endpoint, isWebSocket),
		logContext,
	)
}

// locate attempts to use the backend locator to find an endpoint for the given
// request.
func (handler *Handler) locate(request *http.Request) (*backend.Endpoint, error) {
	serverName, err := name.FromHTTP(request)
	if err != nil {
		return nil, statuspage.Error{
			Inner:      err,
			StatusCode: http.StatusNotFound,
		}
	}

	endpoint := handler.Locator.Locate(request.Context(), serverName)
	if endpoint == nil {
		return nil, statuspage.Error{
			Inner:      errors.New("could not locate backend"),
			StatusCode: http.StatusNotFound,
		}
	}

	return endpoint, nil
}

// prepareUpstreamRequest makes a new http.Request that uses the given endpoint
// as the upstream server.
func (handler *Handler) prepareUpstreamRequest(
	request *http.Request,
	endpoint *backend.Endpoint,
	isWebSocket bool,
) *http.Request {
	upstreamRequest := *request
	upstreamRequest.Header = handler.prepareUpstreamHeaders(request, isWebSocket)

	upstreamURL := *request.URL
	upstreamURL.Host = endpoint.Address

	if isWebSocket {
		if endpoint.TLSMode == backend.TLSDisabled {
			upstreamURL.Scheme = "ws"
		} else {
			upstreamURL.Scheme = "wss"
		}
	} else {
		if endpoint.TLSMode == backend.TLSDisabled {
			upstreamURL.Scheme = "http"
		} else {
			upstreamURL.Scheme = "https"
		}
	}

	upstreamRequest.URL = &upstreamURL

	return &upstreamRequest
}

// prepareUpstreamHeaders produces a copy of request.Header and modifies them so
// that they are suitable to send to the upstream server.
func (handler *Handler) prepareUpstreamHeaders(request *http.Request, isWebSocket bool) http.Header {
	upstreamHeaders := http.Header{}
	forwardedFor, _, _ := net.SplitHostPort(request.RemoteAddr)

	for name, values := range request.Header {
		if name == "X-Forwarded-For" {
			forwardedFor = strings.Join(values, ", ") + ", " + forwardedFor
		} else if !isHopByHopHeader(name) {
			upstreamHeaders[name] = values
		}
	}

	upstreamHeaders.Set("Host", request.Host)
	upstreamHeaders.Set("X-Forwarded-For", forwardedFor)
	upstreamHeaders.Set("X-Forwarded-SSL", "on")

	if isWebSocket {
		upstreamHeaders.Set("X-Forwarded-Proto", "wss")
	} else {
		upstreamHeaders.Set("X-Forwarded-Proto", "https")
	}

	return upstreamHeaders
}

// selectProxy returns the proxy used to connect to the given endpoint.
func (handler *Handler) selectProxy(endpoint *backend.Endpoint, isWebSocket bool) Proxy {
	if endpoint.TLSMode == backend.TLSInsecure {
		if isWebSocket {
			return handler.InsecureWebSocketProxy
		}

		return handler.InsecureHTTPProxy
	}

	if isWebSocket {
		return handler.SecureWebSocketProxy
	}

	return handler.SecureHTTPProxy
}

// writeStatusPage responds with a status page for the given error.
func (handler *Handler) writeStatusPage(
	writer http.ResponseWriter,
	request *http.Request,
	logContext *LogContext,
	err error,
) {
	statusWriter := handler.StatusPageWriter
	if statusWriter == nil {
		statusWriter = statuspage.DefaultWriter
	}

	logContext.Metrics.FirstByteSent()
	defer logContext.Metrics.LastByteSent()

	logContext.StatusCode, logContext.Metrics.BytesOut, _ = statusWriter.WriteError(
		writer,
		request,
		err,
	)
}
