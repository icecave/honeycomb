package di

import "github.com/icecave/honeycomb/src/cert"

// CertificateProvider returns the provider used to load TLS certificates for
// incoming HTTPS requests.
func (con *Container) CertificateProvider() cert.Provider {
	return con.get(
		"cert.provider",
		func() (interface{}, error) {
			return &cert.AdhocProvider{}, nil
		},
		nil,
	).(cert.Provider)
}
