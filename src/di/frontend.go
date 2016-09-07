package di

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/di/container"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"github.com/icecave/honeycomb/src/frontend/health"
	"github.com/icecave/honeycomb/src/proxy"
	"github.com/icecave/honeycomb/src/transaction"
)

func init() {
	Container.DefineEnv("FRONTEND_ADDRESS", ":8443")

	Container.Define("frontend.server", func(d *container.Definer) (interface{}, error) {
		return &http.Server{
			Addr:      d.Get("FRONTEND_ADDRESS").(string),
			TLSConfig: d.Get("frontend.tls-config").(*tls.Config),
			Handler:   d.Get("frontend.http-handler").(http.Handler),
			ErrorLog:  d.Get("logger").(*log.Logger),
		}, nil
	})

	Container.Define("frontend.tls-config", func(d *container.Definer) (interface{}, error) {
		httpHandler := d.Get("frontend.http-handler").(*frontend.HandlerAdaptor)
		certProvider := &cert.ProviderAdaptor{
			PrimaryProvider:   d.Get("frontend.cert.primary-provider").(cert.Provider),
			SecondaryProvider: d.Get("frontend.cert.secondary-provider").(cert.Provider),
			IsRecognised:      httpHandler.IsRecognised,
		}

		return &tls.Config{
			NextProtos:     []string{"h2"},
			GetCertificate: certProvider.GetCertificate,
		}, nil
	})

	Container.Define("frontend.http-handler", func(d *container.Definer) (interface{}, error) {
		return &frontend.HandlerAdaptor{
			Locator:     d.Get("backend.locator").(backend.Locator),
			Handler:     d.Get("frontend.proxy").(transaction.Handler),
			Interceptor: &health.Interceptor{},
			Logger:      d.Get("logger").(*log.Logger),
		}, nil
	})

	Container.Define("frontend.proxy", func(d *container.Definer) (interface{}, error) {
		logger := d.Get("logger").(*log.Logger)
		return &proxy.Proxy{
			HTTPProxy:      proxy.NewHTTPProxy(logger),
			WebSocketProxy: proxy.NewWebSocketProxy(logger),
		}, nil
	})

	Container.Define("frontend.cert.primary-provider", func(d *container.Definer) (interface{}, error) {
		certificatePath := os.Getenv("CERTIFICATE_PATH")

		issuerCertificate, err := cert.LoadX509Certificate(path.Join(certificatePath, "ca.crt"))
		if err != nil {
			return nil, err
		}

		issuerKey, err := cert.LoadPrivateKey(path.Join(certificatePath, "ca.key"))
		if err != nil {
			return nil, err
		}

		serverKey, err := cert.LoadPrivateKey(path.Join(certificatePath, "server.key"))
		if err != nil {
			return nil, err
		}

		return &cert.AdhocProvider{
			Generator: &generator.IssuerSignedGenerator{
				IssuerCertificate: issuerCertificate,
				IssuerKey:         issuerKey,
				ServerKey:         serverKey,
			},
			Logger: d.Get("logger").(*log.Logger),
		}, nil
	})

	Container.Define("frontend.cert.secondary-provider", func(d *container.Definer) (interface{}, error) {
		return d.TryGet("frontend.cert.primary-provider")
	})
}
