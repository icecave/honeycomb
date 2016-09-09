package proxy

import (
	"log"
	"net/http"

	"github.com/icecave/honeycomb/src/statuspage"
)

// Handler is an http.Handler that proxies requests to an upstream server.
type Handler struct {
	Router           Router
	HTTPProxy        Proxy
	WebSocketProxy   Proxy
	StatusPageWriter statuspage.Writer
	Logger           *log.Logger
}

// ServeHTTP proxies the request to the appropriate upstream server.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logContext := &LogContext{Logger: handler.Logger, Request: request}
	logContext.Metrics.Start()

	err := handler.forward(writer, request, logContext)

	// If there was an error and no response has been sent, send an error page.
	if err != nil && logContext.StatusCode == 0 {
		handler.statusPage(writer, request, logContext, err)
	}

	logContext.Log(err)
}

// prepareUpstreamRequest produces an an HTTP request that is used to contact
// an upstream server, before it is updated by the router.
func (handler *Handler) prepareRequest(
	request *http.Request,
) (upstreamRequest *http.Request, isWebSocket bool) {
	upstreamRequest = &*request // shallow copy request

	// Deep-copy (and update) the headers ...
	upstreamRequest.Header, isWebSocket = prepareUpstreamHeaders(request)

	// Deep copy the URL, including the .User, since it's a pointer ...
	upstreamRequest.URL = &*upstreamRequest.URL
	if upstreamRequest.URL.User != nil {
		upstreamRequest.URL.User = &*upstreamRequest.URL.User
	}

	return
}

func (handler *Handler) forward(
	writer http.ResponseWriter,
	request *http.Request,
	logContext *LogContext,
) error {
	upstreamRequest, isWebSocket := handler.prepareRequest(request)
	upstreamInfo, err := handler.Router.Route(
		request,
		isWebSocket,
		upstreamRequest.URL,
		upstreamRequest.Header,
	)

	logContext.IsWebSocket = isWebSocket

	if err != nil {
		return err
	}

	logContext.UpstreamRequest = upstreamRequest
	logContext.UpstreamInfo = upstreamInfo

	var proxy Proxy
	if isWebSocket {
		proxy = handler.WebSocketProxy
	} else {
		proxy = handler.HTTPProxy
	}

	return proxy.Forward(
		writer,
		request,
		upstreamRequest,
		logContext,
	)
}

func (handler *Handler) statusPage(
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
