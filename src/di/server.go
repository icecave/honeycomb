package di

import (
	"os"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/frontend"
	"github.com/icecave/honeycomb/src/proxy"
)

// Server returns a new front-end server.
func (con *Container) Server() *frontend.Server {
	return con.get(
		"server",
		func() (interface{}, error) {
			logger := con.Logger()
			return &frontend.Server{
				BindAddress:         con.BindAddress(),
				Locator:             con.Locator(),
				CertificateProvider: con.CertificateProvider(),
				HTTPProxy:           proxy.NewHTTPProxy(logger),
				WebSocketProxy:      proxy.NewWebSocketProxy(logger),
				Logger:              logger,
			}, nil
		},
		nil,
	).(*frontend.Server)
}

// BindAddress returns the address that the server should listen on.
func (con *Container) BindAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		return ":8443"
	}
	return ":" + port
}

// Locator returns the back-end locator used to resolve domain names to servers.
func (con *Container) Locator() backend.Locator {
	return con.get(
		"server.locator",
		func() (interface{}, error) {
			locator := &backend.StaticLocator{}
			locator.Add(
				"foo.lvh.me",
				&backend.Endpoint{
					Address: "localhost:8080",
					IsTLS:   false,
				},
			)
			return locator, nil
		},
		nil,
	).(backend.Locator)
}
