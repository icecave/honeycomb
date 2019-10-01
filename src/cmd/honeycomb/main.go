package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/docker"
	"github.com/icecave/honeycomb/src/docker/health"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/proxy"
	"github.com/icecave/honeycomb/src/proxyprotocol"
	"github.com/icecave/honeycomb/src/static"
	"go.uber.org/multierr"
)

var version = "notset"

func main() {
	config := cmd.GetConfigFromEnvironment()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	staticLocator, err := static.FromEnv(logger)
	if err != nil {
		logger.Fatalln(err)
	}

	dockerClient, err := client.NewClientWithOpts(dockerClientFromEnvironment)
	if err != nil {
		logger.Fatalln(err)
	}

	cachingLocator := &backend.Cache{}

	dockerLocator := &docker.Locator{
		Loader: &docker.ServiceLoader{
			Client: dockerClient,
			Inspector: &docker.ServiceInspector{
				Client: dockerClient,
			},
			Logger: logger,
		},
		Cache:  cachingLocator,
		Logger: logger,
	}
	go dockerLocator.Run()
	defer dockerLocator.Stop()

	cachingLocator.Next = backend.AggregateLocator{
		staticLocator,
		dockerLocator,
	}

	defaultCertificate, err := loadDefaultCertificate(config)
	if err != nil {
		logger.Fatalln(err)
	}

	rootCACertPool := rootCAPool(config, logger)

	resolver := certificateResolver(
		config,
		cachingLocator,
		defaultCertificate.PrivateKey.(*rsa.PrivateKey),
		logger,
	)

	tlsConfig := &tls.Config{
		GetCertificate: resolver.GetCertificate,
		Certificates:   []tls.Certificate{*defaultCertificate},
		RootCAs:        rootCACertPool,
	}

	secureTransport := &http.Transport{
		Proxy:                 http.DefaultTransport.(*http.Transport).Proxy,
		DialContext:           http.DefaultTransport.(*http.Transport).DialContext,
		MaxIdleConns:          http.DefaultTransport.(*http.Transport).MaxIdleConns,
		IdleConnTimeout:       http.DefaultTransport.(*http.Transport).IdleConnTimeout,
		TLSHandshakeTimeout:   http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout,
		ExpectContinueTimeout: http.DefaultTransport.(*http.Transport).ExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			RootCAs: rootCACertPool,
		},
	}

	insecureTransport := &http.Transport{
		Proxy:                 http.DefaultTransport.(*http.Transport).Proxy,
		DialContext:           http.DefaultTransport.(*http.Transport).DialContext,
		MaxIdleConns:          http.DefaultTransport.(*http.Transport).MaxIdleConns,
		IdleConnTimeout:       http.DefaultTransport.(*http.Transport).IdleConnTimeout,
		TLSHandshakeTimeout:   http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout,
		ExpectContinueTimeout: http.DefaultTransport.(*http.Transport).ExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCACertPool,
			InsecureSkipVerify: true,
		},
	}

	prepareTLSConfig(tlsConfig)

	server := http.Server{
		Addr:      ":" + config.Port,
		TLSConfig: tlsConfig,
		Handler: &frontend.Handler{
			Proxy: &proxy.Handler{
				Locator: cachingLocator,
				SecureHTTPProxy: &proxy.HTTPProxy{
					Transport: secureTransport,
				},
				InsecureHTTPProxy: &proxy.HTTPProxy{
					Transport: insecureTransport,
				},
				SecureWebSocketProxy: &proxy.WebSocketProxy{
					Dialer: &proxy.BasicWebSocketDialer{
						TLSConfig: secureTransport.TLSClientConfig,
					},
				},
				InsecureWebSocketProxy: &proxy.WebSocketProxy{
					Dialer: &proxy.BasicWebSocketDialer{
						TLSConfig: secureTransport.TLSClientConfig,
					},
				},
				Logger: logger,
			},
			HealthCheck: &health.HTTPHandler{
				Checker: &health.SwarmChecker{
					Client: dockerClient,
				},
				Logger: logger,
			},
			Logger: logger,
		},
		ErrorLog: logger,
	}

	go redirectServer(config, logger)

	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Fatal(err)
	}

	if config.ProxyProtocol {
		listener = proxyprotocol.NewListener(listener)
	}

	logger.Printf("Listening on port %s", config.Port)

	err = server.ServeTLS(listener, "", "")
	if err != nil {
		logger.Fatalln(err)
	}
}

func dockerClientFromEnvironment(c *client.Client) error {
	return multierr.Append(
		client.FromEnv(c),
		client.WithHTTPHeaders(
			map[string]string{
				"User-Agent": fmt.Sprintf("Honeycomb/%s", version),
			},
		)(c),
	)
}

func loadDefaultCertificate(config *cmd.Config) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(
		path.Join(config.Certificates.BasePath, config.Certificates.ServerCertificate),
		path.Join(config.Certificates.BasePath, config.Certificates.ServerKey),
	)
	if err != nil {
		return nil, err
	}
	issuer, err := tls.LoadX509KeyPair(
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerCertificate),
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerKey),
	)
	if err != nil {
		return nil, err
	}
	cert.Certificate = append(cert.Certificate, issuer.Certificate...)
	return &cert, err
}

func prepareTLSConfig(config *tls.Config) {
	config.NextProtos = []string{"h2"}
	config.MinVersion = tls.VersionTLS10
	config.PreferServerCipherSuites = true
	config.CurvePreferences = []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521}
}

func rootCAPool(
	config *cmd.Config,
	logger *log.Logger,
) *x509.CertPool {
	pool := x509.NewCertPool()
	count := len(pool.Subjects())

	for _, filename := range config.Certificates.CABundles {
		buf, err := ioutil.ReadFile(filename)
		if err == nil {
			pool.AppendCertsFromPEM(buf)
			c := len(pool.Subjects())
			logger.Printf("Loaded %d certificate(s) from CA bundle at %s", c-count, filename)
			count = c
		} else if !os.IsNotExist(err) {
			logger.Fatalln(err)
		}
	}

	return pool
}

func redirectServer(config *cmd.Config, logger *log.Logger) {
	listener, err := net.Listen("tcp", ":"+config.InsecurePort)
	if err != nil {
		logger.Fatal(err)
	}

	if config.ProxyProtocol {
		listener = proxyprotocol.NewListener(listener)
	}

	http.Serve(
		listener,
		http.HandlerFunc(redirectHandler),
	)
}

func redirectHandler(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}
