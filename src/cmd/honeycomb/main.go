package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/docker"
	"github.com/icecave/honeycomb/src/docker/health"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"github.com/icecave/honeycomb/src/proxy"
	"github.com/icecave/honeycomb/src/static"
)

func main() {
	config := cmd.GetConfigFromEnvironment()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	staticLocator, err := static.FromEnv(logger)
	if err != nil {
		logger.Fatalln(err)
	}

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		logger.Fatalln(err)
	}

	dockerLocator := &docker.Locator{
		Loader: &docker.ServiceLoader{
			Client: dockerClient,
			Inspector: &docker.ServiceInspector{
				Client: dockerClient,
			},
			Logger: logger,
		},
		Logger: logger,
	}
	go dockerLocator.Run()
	defer dockerLocator.Stop()

	locator := backend.AggregateLocator{
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
		PrimaryProvider:   primaryCertificateProvider(config, logger),
		SecondaryProvider: secondaryCertProvider,
	}

	tlsConfig := &tls.Config{
		GetCertificate: providerAdaptor.GetCertificate,
		Certificates:   []tls.Certificate{*defaultCertificate},
	}

	prepareTLSConfig(tlsConfig)

	server := http.Server{
		Addr:      ":" + config.Port,
		TLSConfig: tlsConfig,
		Handler: &frontend.Handler{
			Proxy: &proxy.Handler{
				Locator:        locator,
				HTTPProxy:      &proxy.HTTPProxy{},
				WebSocketProxy: &proxy.WebSocketProxy{},
				Logger:         logger,
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

	go redirectServer(config)

	logger.Printf("Listening on port %s", config.Port)

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		logger.Fatalln(err)
	}
}

func loadDefaultCertificate(config *cmd.Config) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(
		path.Join(config.Certificates.BasePath, config.Certificates.ServerCertificate),
		path.Join(config.Certificates.BasePath, config.Certificates.ServerKey),
	)
	if err != nil {
		return nil, err
	}
	return &cert, err
}

func primaryCertificateProvider(
	config *cmd.Config,
	logger *log.Logger,
) cert.Provider {
	return &cert.FileProvider{
		BasePath: config.Certificates.BasePath,
		Logger:   logger,
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

func prepareTLSConfig(config *tls.Config) {
	config.NextProtos = []string{"h2"}
	config.MinVersion = tls.VersionTLS10
	config.PreferServerCipherSuites = true
	config.CurvePreferences = []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521}
}

func redirectServer(config *cmd.Config) {
	http.ListenAndServe(
		":"+config.InsecurePort,
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
