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
	Interceptor         Interceptor
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
	ctx := &RequestContext{}
	ctx.Timer.MarkReceived()

	ctx.Request = request
	ctx.Writer.Inner = innerWriter
	ctx.Writer.OnRespond = ctx.Timer.MarkResponded
	ctx.Writer.OnHijack = func() {
		ctx.Timer.MarkResponded()
		ctx.Log(svr.Logger)
	}

	ctx.IsWebSocket = websocket.IsWebSocketUpgrade(request)
	ctx.ServerName, ctx.Error = serverNameFromRequest(request)

	// The server name was normalized successfully ...
	if ctx.Error == nil {
		ctx.Endpoint = svr.Locator.Locate(request.Context(), ctx.ServerName)

		if ctx.Endpoint == nil {
			ctx.Error = fmt.Errorf("can not locate back-end for '%s'", ctx.ServerName.Unicode)
		}

		if svr.Interceptor != nil {
			svr.Interceptor.Intercept(ctx)
		}

		if !ctx.Intercepted {
			svr.Metrics.StartRequest(ctx)
			defer svr.Metrics.EndRequest(ctx)

			if ctx.Error != nil {
				proxy.WriteError(&ctx.Writer, http.StatusServiceUnavailable)
			} else if ctx.IsWebSocket {
				ctx.Error = svr.WebSocketProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
			} else {
				ctx.Error = svr.HTTPProxy.ForwardRequest(ctx.Endpoint, &ctx.Writer, ctx.Request)
			}
		}
	}

	ctx.Timer.MarkCompleted()
	ctx.Log(svr.Logger)
}

func (svr *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if info.ServerName == "" {
		return nil, fmt.Errorf("no SNI information")
	}

	serverName, err := name.TryParseServerName(info.ServerName)
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

	// If we can't find one, make sure there is an endpoint to route to ...
	if endpoint := svr.Locator.Locate(ctx, serverName); endpoint != nil {
		return svr.CertificateProvider.GetCertificate(ctx, serverName)

	}

	// Or otherwise that the interceptor provides the services ...
	if svr.Interceptor != nil && svr.Interceptor.Provides(serverName) {
		return svr.CertificateProvider.GetCertificate(ctx, serverName)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("can not locate back-end for '%s'", serverName.Unicode)
}

func serverNameFromRequest(request *http.Request) (name.ServerName, error) {
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host
	}

	return name.TryParseServerName(host)
}
