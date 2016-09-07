package di

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
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
	"github.com/icecave/honeycomb/src/request"
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
			Handler:     d.Get("frontend.proxy").(request.Handler),
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
		certificate, err := tls.LoadX509KeyPair(
			path.Join(certificatePath, "ca.crt"),
			path.Join(certificatePath, "ca.key"),
		)
		if err != nil {
			return nil, err
		}

		raw, err := ioutil.ReadFile(
			path.Join(certificatePath, "server.key"),
		)
		if err != nil {
			return nil, err
		}

		block, unused := pem.Decode(raw)
		if len(unused) > 0 {
			return nil, err
		}

		serverKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return &cert.AdhocProvider{
			Generator: &generator.IssuerSignedGenerator{
				IssuerCertificate: certificate.Leaf,
				IssuerKey:         certificate.PrivateKey,
				ServerKey:         serverKey,
			},
			Logger: d.Get("logger").(*log.Logger),
		}, nil
	})

	Container.Define("frontend.cert.secondary-provider", func(d *container.Definer) (interface{}, error) {
		return d.TryGet("frontend.cert.primary-provider")
	})
}
