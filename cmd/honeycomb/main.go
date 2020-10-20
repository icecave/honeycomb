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

	"golang.org/x/net/http2"

	"github.com/docker/docker/client"
	"github.com/go-redis/redis/v8"
	"github.com/icecave/honeycomb/backend"
	"github.com/icecave/honeycomb/cmd"
	"github.com/icecave/honeycomb/docker"
	"github.com/icecave/honeycomb/docker/health"
	"github.com/icecave/honeycomb/frontend"
	"github.com/icecave/honeycomb/frontend/cert"
	"github.com/icecave/honeycomb/frontend/cert/generator"
	"github.com/icecave/honeycomb/proxy"
	"github.com/icecave/honeycomb/proxyprotocol"
	"github.com/icecave/honeycomb/static"
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

	secondaryCertProvider, err := secondaryCertificateProvider(
		config,
		defaultCertificate.PrivateKey.(*rsa.PrivateKey),
		logger,
	)
	if err != nil {
		logger.Fatalln(err)
	}

	providerAdaptor := &cert.ProviderAdaptor{
		PrimaryProvider: &cert.MultiProvider{
			Providers: []cert.Provider{
				primaryFileCertificateProvider(config, logger),
				primaryRedisCertificateProvider(config, logger),
			},
		},
		SecondaryProvider: secondaryCertProvider,
	}

	rootCACertPool := rootCAPool(config, logger)

	tlsConfig := &tls.Config{
		GetCertificate: providerAdaptor.GetCertificate,
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

	h2cTransport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}

	prepareTLSConfig(config, tlsConfig)

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
				H2CProxy: &proxy.HTTPProxy{
					Transport: h2cTransport,
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

func primaryFileCertificateProvider(
	config *cmd.Config,
	logger *log.Logger,
) cert.Provider {
	return &cert.FileProvider{
		BasePath: config.Certificates.BasePath,
	}
}

func primaryRedisCertificateProvider(
	config *cmd.Config,
	logger *log.Logger,
) cert.Provider {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Certificates.RedisAddress,
		Password: config.Certificates.RedisPassword,
	})

	return &cert.RedisProvider{
		Client:   rdb,
		Logger:   logger,
		CacheAge: config.Certificates.RedisCacheExpire,
	}
}

func secondaryCertificateProvider(
	config *cmd.Config,
	serverKey *rsa.PrivateKey,
	logger *log.Logger,
) (cert.Provider, error) {
	issuer, err := tls.LoadX509KeyPair(
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerCertificate),
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerKey),
	)
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(issuer.Certificate[0])
	if err != nil {
		return nil, err
	}

	issuer.Leaf = x509Cert

	return &cert.AdhocProvider{
		Generator: &generator.IssuerSignedGenerator{
			IssuerCertificate: issuer.Leaf,
			IssuerKey:         issuer.PrivateKey.(*rsa.PrivateKey),
			ServerKey:         serverKey,
		},
		Logger: logger,
	}, nil
}

func prepareTLSConfig(config *cmd.Config, tlsConfig *tls.Config) {
	tlsConfig.NextProtos = []string{"h2"}
	tlsConfig.MinVersion = config.MinTLSVersion
	tlsConfig.MaxVersion = config.MaxTLSVersion
	tlsConfig.CipherSuites = config.CipherSuite
	tlsConfig.PreferServerCipherSuites = true
	tlsConfig.CurvePreferences = []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521}
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
