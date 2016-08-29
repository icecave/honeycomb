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

// Server handles incoming web requests.
type Server interface {
	// Run starts the server and blocks until it is finished.
	Run() error
}

// NewServer creates a new HTTPS server.
func NewServer(
	bindAddress string,
	locator backend.Locator,
	certProvider cert.Provider,
	httpProxy proxy.Proxy,
	websocketProxy proxy.Proxy,
	logger *log.Logger,
) Server {
	return &server{
		bindAddress,
		locator,
		certProvider,
		httpProxy,
		websocketProxy,
		logger,
	}
}

type server struct {
	bindAddress    string
	locator        backend.Locator
	certProvider   cert.Provider
	httpProxy      proxy.Proxy
	websocketProxy proxy.Proxy
	logger         *log.Logger
}

func (svr *server) Run() error {
	tlsConfig := &tls.Config{
		GetCertificate: svr.getCertificate,
		NextProtos:     []string{"h2"}, // explicitly enable HTTP/2
	}

	listener, err := tls.Listen("tcp", svr.bindAddress, tlsConfig)
	if err != nil {
		return err
	}

	httpServer := http.Server{
		TLSConfig: tlsConfig,
		Handler:   http.HandlerFunc(svr.serve),
		ErrorLog:  svr.logger,
	}

	svr.logger.Printf("listening on %s", svr.bindAddress)

	return httpServer.Serve(listener)
}

func (svr *server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// Make sure we can locate a back-end for the domain before we request a
	// certificate for it ...
	if svr.locator.CanLocate(info.ServerName) {
		return svr.certProvider.GetCertificate(info)
	}

	// Ideally we would return an "unrecognised_name" TLS alert here, but Go's
	// HTTP server has no way to do so, so let it fail with an "internal_error" ...
	return nil, fmt.Errorf("back-end for '%s' does not exist", info.ServerName)
}

func (svr *server) serve(response http.ResponseWriter, request *http.Request) {
	domainName, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		domainName = request.Host
	}

	endpoint, ok := svr.locator.Locate(domainName)

	if !ok {
		svr.logger.Printf(
			"%s | %s | back-end went away after TLS negotiation",
			request.RemoteAddr,
			domainName,
		)
		http.Error(response, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	isWebSocket := websocket.IsWebSocketUpgrade(request)
	var prx proxy.Proxy
	var method string

	if isWebSocket {
		prx = svr.websocketProxy
		method = "WEBSOCK"
	} else {
		prx = svr.httpProxy
		method = request.Method
	}

	svr.logger.Printf(
		"%s | %s -> %s | %s %s",
		request.RemoteAddr,
		domainName,
		endpoint.Address,
		method,
		request.URL,
	)

	err = prx.ForwardRequest(endpoint, response, request)

	if err != nil {
		svr.logger.Printf(
			"%s | %s -> %s | %s",
			request.RemoteAddr,
			domainName,
			endpoint.Address,
			err,
		)
	}
}
