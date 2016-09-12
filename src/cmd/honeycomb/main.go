package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/docker/engine-api/client"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/docker"
	"github.com/icecave/honeycomb/src/docker/health"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"github.com/icecave/honeycomb/src/proxy"
)

func main() {
	config := cmd.GetConfigFromEnvironment()
	logger := log.New(os.Stdout, "", log.LstdFlags)

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

	certProvider, err := certificateProvider(logger, config)
	if err != nil {
		logger.Fatalln(err)
	}

	providerAdaptor := &cert.ProviderAdaptor{
		PrimaryProvider:   certProvider,
		SecondaryProvider: certProvider,
	}

	server := http.Server{
		Addr: ":" + config.Port,
		TLSConfig: &tls.Config{
			NextProtos:     []string{"h2"},
			GetCertificate: providerAdaptor.GetCertificate,
		},
		Handler: &frontend.Handler{
			Proxy: &proxy.Handler{
				Locator: backend.AggregateLocator{
					backend.StaticLocator{}.With(
						"static.lvh.me",
						&backend.Endpoint{
							Address:     "localhost:8080",
							Description: "local echo server",
						},
					),
					dockerLocator,
				},
				HTTPProxy:      &proxy.HTTPProxy{},
				WebSocketProxy: &proxy.WebSocketProxy{},
				Logger:         logger,
			},
			HealthCheck: &health.HTTPHandler{
				// Checker: Checker, @todo
				Logger: logger,
			},
			Logger: logger,
		},
		ErrorLog: logger,
	}

	logger.Printf("Listening on port %s", config.Port)
	err = server.ListenAndServeTLS(config.ServerCertificate, config.ServerKey)
	if err != nil {
		logger.Fatalln(err)
	}
}

func certificateProvider(logger *log.Logger, config *cmd.Config) (cert.Provider, error) {
	issuerCertificate, err := cert.LoadX509Certificate(config.CACertificate)
	if err != nil {
		return nil, err
	}

	issuerKey, err := cert.LoadPrivateKey(config.CAKey)
	if err != nil {
		return nil, err
	}

	serverKey, err := cert.LoadPrivateKey(config.ServerKey)
	if err != nil {
		return nil, err
	}

	return &cert.AdhocProvider{
		Generator: &generator.IssuerSignedGenerator{
			IssuerCertificate: issuerCertificate,
			IssuerKey:         issuerKey,
			ServerKey:         serverKey,
		},
		Logger: logger,
	}, nil
}
