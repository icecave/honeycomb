package frontend

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

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

	writer := &proxy.ResponseWriter{
		Inner:      innerWriter,
		FirstWrite: timer.MarkResponded,
	}

	endpoint, err := svr.locateBackend(request)
	isWebSocket := websocket.IsWebSocketUpgrade(request)

	if err != nil {
		http.Error(writer, "Service Unavailable", http.StatusServiceUnavailable)
	} else if isWebSocket {
		svr.logRequest(endpoint, writer, request, timer, isWebSocket, nil)
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

	endpoint, ok := svr.Locator.Locate(domainName)
	if ok {
		return endpoint, nil
	}

	return nil, fmt.Errorf("can not locate back-end for '%s'", domainName)
}

func (svr *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if info.ServerName == "" {
		return nil, fmt.Errorf("no SNI information")
	}

	// Make sure we can locate a back-end for the domain before we request a
	// certificate for it ...
	if svr.Locator.CanLocate(info.ServerName) {
		return svr.CertificateProvider.GetCertificate(info)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("can not locate back-end for '%s'", info.ServerName)
}

func (svr *Server) logRequest(
	endpoint *backend.Endpoint,
	writer *proxy.ResponseWriter,
	request *http.Request,
	timer *requestTimer,
	isWebSocket bool,
	err error,
) {
	backend := "-"
	if endpoint != nil {
		backend = endpoint.Address
	}

	method := request.Method
	if isWebSocket {
		method = "WEBSOCK"
	}

	message := fmt.Sprintf(
		"http: [%s] %s %s \"%s %s %s\"",
		request.RemoteAddr,
		request.Host,
		backend,
		method,
		request.URL,
		request.Proto,
	)

	// A response has been received ...
	if writer.StatusCode != 0 {
		message += fmt.Sprintf(
			" %d %d +%s +%s",
			writer.StatusCode,
			writer.Size,
			timer.TimeToFirstByte(),
			timer.TransmissionTime(),
		)
	}

	if err != nil {
		message += " " + err.Error()
	}

	svr.Logger.Println(message)
}
