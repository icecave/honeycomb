package frontend

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"golang.org/x/net/idna"

	"github.com/gorilla/websocket"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/cert"
	"github.com/icecave/honeycomb/src/proxy"
)

// Server listens for "front-end" TLS connections from external clients.
type Server struct {
	BindAddress         string
	Locator             backend.Locator
	CertificateProvider cert.Provider
	HTTPProxy           proxy.Proxy
	WebSocketProxy      proxy.Proxy
	Logger              *log.Logger
	Metrics             Metrics
}

// Run starts the server and blocks until it exits.
func (svr *Server) Run() error {
	tlsConfig := &tls.Config{
		GetCertificate: svr.getCertificate,
		NextProtos:     []string{"h2"}, // explicitly enable HTTP/2
	}

	listener, err := tls.Listen("tcp", svr.BindAddress, tlsConfig)
	if err != nil {
		return err
	}

	httpServer := http.Server{
		TLSConfig: tlsConfig,
		Handler:   http.HandlerFunc(svr.forwardRequest),
		ErrorLog:  svr.Logger,
	}

	svr.Logger.Printf("frontend: Listening on %s", svr.BindAddress)
	return httpServer.Serve(listener)
}

// forwardRequest is the server's internal request handler
func (svr *Server) forwardRequest(innerWriter http.ResponseWriter, request *http.Request) {
	var ctx requestContext
	ctx.Request = request
	ctx.Timer.MarkReceived()
	ctx.IsWebSocket = websocket.IsWebSocketUpgrade(request)
	ctx.Endpoint, ctx.Error = svr.locateBackend(request)

	ctx.Writer.Inner = innerWriter
	ctx.Writer.OnRespond = ctx.Timer.MarkResponded
	ctx.Writer.OnHijack = func() {
		ctx.Timer.MarkResponded()
		svr.logRequest(&ctx)
	}

	svr.Metrics.StartRequest(&ctx)

	if ctx.Error != nil {
		http.Error(&ctx.Writer, "Service Unavailable", http.StatusServiceUnavailable)
	} else if ctx.IsWebSocket {
		ctx.Error = svr.WebSocketProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
	} else {
		ctx.Error = svr.HTTPProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
	}

	ctx.Timer.MarkCompleted()
	svr.logRequest(&ctx)
	svr.Metrics.EndRequest(&ctx)
}

func (svr *Server) locateBackend(request *http.Request) (*backend.Endpoint, error) {
	serverName, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		serverName = request.Host
	}

	endpoint := svr.Locator.Locate(request.Context(), serverName)
	if endpoint == nil {
		return nil, fmt.Errorf("can not locate back-end for '%s'", serverName)
	}

	return endpoint, nil
}

func (svr *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	serverName, err := idna.ToUnicode(info.ServerName)
	if err != nil {
		return nil, err
	} else if serverName == "" {
		return nil, fmt.Errorf("no SNI information")
	}

	// Make sure we can locate a back-end for the server name before we request
	// a certificate for it ...
	if endpoint := svr.Locator.Locate(context.TODO(), info.ServerName); endpoint != nil {
		return svr.CertificateProvider.GetCertificate(info)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("can not locate back-end for '%s'", serverName)
}

func (svr *Server) logRequest(ctx *requestContext) {
	frontend := ""
	backend := "- -"
	statusCode := 0
	responseSize := "-"
	timeToFirstByte := "-"
	totalTime := "-"
	info := ""

	if ctx.Error != nil {
		info = ctx.Error.Error()
	}

	if ctx.IsWebSocket {
		frontend = "wss://" + ctx.Request.Host
		statusCode = http.StatusSwitchingProtocols
		responseSize = "-"
	} else {
		frontend = "https://" + ctx.Request.Host
		statusCode = ctx.Writer.StatusCode
		responseSize = strconv.Itoa(ctx.Writer.Size)
	}

	// @todo use endpoint.Name in the logs somewhere
	if ctx.Endpoint != nil {
		backend = fmt.Sprintf(
			"%s://%s (%s)",
			ctx.Endpoint.GetScheme(ctx.IsWebSocket),
			ctx.Endpoint.Address,
			ctx.Endpoint.Description,
		)
	}

	if ctx.Timer.HasResponded() {
		timeToFirstByte = ctx.Timer.TimeToFirstByte().String()

		if ctx.Timer.IsComplete() {
			totalTime = ctx.Timer.TimeToLastByte().String()
		} else if ctx.IsWebSocket && info == "" {
			info = "connection established"
		}
	}

	if info != "" {
		info = fmt.Sprintf(" (%s)", info)
	}

	svr.Logger.Printf(
		"frontend: %s %s %s \"%s %s %s\" %d %s %s %s%s",
		ctx.Request.RemoteAddr,
		frontend,
		backend,
		ctx.Request.Method,
		ctx.Request.URL,
		ctx.Request.Proto,
		statusCode,
		responseSize,
		timeToFirstByte,
		totalTime,
		info,
	)
}
