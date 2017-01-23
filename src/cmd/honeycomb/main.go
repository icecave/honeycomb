package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/docker"
	"github.com/icecave/honeycomb/src/docker/health"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"github.com/icecave/honeycomb/src/frontend/cert/loader"
	"github.com/icecave/honeycomb/src/proxy"
	minio "github.com/minio/minio-go"
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

	certLoader, err := certificateLoader(logger, config)
	if err != nil {
		logger.Fatalln(err)
	}

	defaultCertificate, err := loader.LoadX509KeyPair(
		context.Background(),
		certLoader,
		config.Certificates.ServerCertificate,
		config.Certificates.ServerKey,
	)
	if err != nil {
		logger.Fatalln(err)
	}

	certProvider, err := certificateProvider(
		logger,
		certLoader,
		defaultCertificate.PrivateKey.(*rsa.PrivateKey),
		config,
	)
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
			Certificates:   []tls.Certificate{*defaultCertificate},
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
				Checker: &health.SwarmChecker{
					Client: dockerClient,
				},
				Logger: logger,
			},
			Logger: logger,
		},
		ErrorLog: logger,
	}

	logger.Printf("Listening on port %s", config.Port)

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		logger.Fatalln(err)
	}
}

func certificateLoader(
	logger *log.Logger,
	config *cmd.Config,
) (loader.Loader, error) {
	if config.Certificates.S3Bucket == "" {
		return &loader.FileLoader{
			BasePath: config.Certificates.BasePath,
		}, nil
	}

	s3client, err := minio.New(
		config.Certificates.S3Endpoint,
		config.AWSAccessKeyID,
		config.AWSSecretAccessKey,
		true, // secure
	)
	if err != nil {
		return nil, err
	}

	return &loader.S3Loader{
		Bucket:   config.Certificates.S3Bucket,
		BasePath: config.Certificates.BasePath,
		S3Client: s3client,
		Logger:   logger,
	}, nil
}

func certificateProvider(
	logger *log.Logger,
	loader loader.Loader,
	serverKey *rsa.PrivateKey,
	config *cmd.Config,
) (cert.Provider, error) {
	issuerCertificate, err := loader.LoadCertificate(
		context.Background(),
		config.Certificates.IssuerCertificate,
	)
	if err != nil {
		return nil, err
	}

	issuerKey, err := loader.LoadPrivateKey(
		context.Background(),
		config.Certificates.IssuerKey,
	)
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
