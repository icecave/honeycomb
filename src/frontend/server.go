package frontend

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/name"
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
		svr.Logger.Printf("frontend: %s", err)
		return err
	}

	httpServer := http.Server{
		TLSConfig: tlsConfig,
		Handler:   http.HandlerFunc(svr.forwardRequest),
		ErrorLog:  svr.Logger,
	}

	svr.Logger.Printf("frontend: Listening on %s", svr.BindAddress)
	err = httpServer.Serve(listener)
	if err != nil {
		svr.Logger.Printf("frontend: %s", err)
		return err
	}

	return nil
}

// forwardRequest is the server's internal request handler
func (svr *Server) forwardRequest(innerWriter http.ResponseWriter, request *http.Request) {
	ctx := &requestContext{}
	ctx.Timer.MarkReceived()
	ctx.Request = request
	ctx.IsWebSocket = websocket.IsWebSocketUpgrade(request)
	ctx.Endpoint, ctx.Error = svr.locateBackend(request)

	ctx.Writer.Inner = innerWriter
	ctx.Writer.OnRespond = ctx.Timer.MarkResponded
	ctx.Writer.OnHijack = func() {
		ctx.Timer.MarkResponded()
		svr.Logger.Printf("frontend: %s", ctx)
	}

	svr.Metrics.StartRequest(ctx)

	if ctx.Error != nil {
		http.Error(&ctx.Writer, "Service Unavailable", http.StatusServiceUnavailable)
	} else if ctx.IsWebSocket {
		ctx.Error = svr.WebSocketProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
	} else {
		ctx.Error = svr.HTTPProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
	}

	ctx.Timer.MarkCompleted()
	svr.Logger.Printf("frontend: %s", ctx)
	svr.Metrics.EndRequest(ctx)
}

func (svr *Server) locateBackend(request *http.Request) (*backend.Endpoint, error) {
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host
	}

	serverName, err := name.TryNormalizeServerName(host)
	if err != nil {
		return nil, err
	}

	endpoint := svr.Locator.Locate(request.Context(), serverName)
	if endpoint == nil {
		return nil, fmt.Errorf("can not locate back-end for '%s'", serverName.Unicode)
	}

	return endpoint, nil
}

func (svr *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if info.ServerName == "" {
		return nil, fmt.Errorf("no SNI information")
	}

	serverName, err := name.TryNormalizeServerName(info.ServerName)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()

	// First try to find an existing certificate ...
	certificate, err := svr.CertificateProvider.GetExistingCertificate(ctx, serverName)
	if err != nil {
		return nil, err
	} else if certificate != nil {
		return certificate, nil
	}

	// If we can't find one, make sure we at least have an endpoint to route to ...
	if endpoint := svr.Locator.Locate(ctx, serverName); endpoint != nil {
		return svr.CertificateProvider.GetCertificate(ctx, serverName)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("can not locate back-end for '%s'", serverName.Unicode)
}
