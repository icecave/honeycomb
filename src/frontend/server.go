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

	svr.Logger.Printf("http: Listening on %s", svr.BindAddress)
	return httpServer.Serve(listener)
}

// forwardRequest is the server's internal request handler
func (svr *Server) forwardRequest(innerWriter http.ResponseWriter, request *http.Request) {
	timer := &requestTimer{}
	timer.MarkReceived()

	endpoint, err := svr.locateBackend(request)
	isWebSocket := websocket.IsWebSocketUpgrade(request)

	writer := &proxy.ResponseWriter{Inner: innerWriter}
	writer.OnRespond = timer.MarkResponded
	writer.OnHijack = func() {
		timer.MarkResponded()
		svr.logRequest(endpoint, writer, request, timer, isWebSocket, nil)
	}

	if err != nil {
		http.Error(writer, "Service Unavailable", http.StatusServiceUnavailable)
	} else if isWebSocket {
		err = svr.WebSocketProxy.ForwardRequest(endpoint, writer, request)
	} else {
		err = svr.HTTPProxy.ForwardRequest(endpoint, writer, request)
	}

	timer.MarkCompleted()
	svr.logRequest(endpoint, writer, request, timer, isWebSocket, err)
}

func (svr *Server) locateBackend(request *http.Request) (*backend.Endpoint, error) {
	domainName, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		domainName = request.Host
	}

	endpoint := svr.Locator.Locate(request.Context(), domainName)
	if endpoint == nil {
		return nil, fmt.Errorf("can not locate back-end for '%s'", domainName)
	}

	return endpoint, nil
}

func (svr *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domainName, err := idna.ToUnicode(info.ServerName)
	if err != nil {
		return nil, err
	} else if domainName == "" {
		return nil, fmt.Errorf("no SNI information")
	}

	// Make sure we can locate a back-end for the domain before we request a
	// certificate for it ...
	if svr.Locator.CanLocate(context.TODO(), info.ServerName) {
		return svr.CertificateProvider.GetCertificate(info)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("can not locate back-end for '%s'", domainName)
}

func (svr *Server) logRequest(
	endpoint *backend.Endpoint,
	writer *proxy.ResponseWriter,
	request *http.Request,
	timer *requestTimer,
	isWebSocket bool,
	info interface{},
) {
	frontend := ""
	backend := "-"
	statusCode := 0
	responseSize := "-"
	timeToFirstByte := "-"
	totalTime := "-"

	if isWebSocket {
		frontend = "wss://" + request.Host
		statusCode = http.StatusSwitchingProtocols
		responseSize = "-"
	} else {
		frontend = "https://" + request.Host
		statusCode = writer.StatusCode
		responseSize = strconv.Itoa(writer.Size)
	}

	// @todo use endpoint.Name in the logs somewhere
	if endpoint != nil {
		backend = fmt.Sprintf(
			"%s://%s",
			endpoint.GetScheme(isWebSocket),
			endpoint.Address,
		)
	}

	if timer.HasResponded() {
		timeToFirstByte = timer.TimeToFirstByte().String()

		if timer.IsComplete() {
			totalTime = timer.TotalTime().String()
		} else if isWebSocket && info == nil {
			info = "connection established"
		}
	}

	if info == nil {
		info = ""
	} else {
		info = fmt.Sprintf(" (%s)", info)
	}

	svr.Logger.Printf(
		"http: %s %s %s \"%s %s %s\" %d %s %s %s%s",
		request.RemoteAddr,
		frontend,
		backend,
		request.Method,
		request.URL,
		request.Proto,
		statusCode,
		responseSize,
		timeToFirstByte,
		totalTime,
		info,
	)
}
