package di

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/icecave/honeycomb/src/cert"
)

// CertificateProvider returns the provider used to load TLS certificates for
// incoming HTTPS requests.
func (con *Container) CertificateProvider() cert.Provider {
	return con.get(
		"cert.provider",
		func() (interface{}, error) {
			return cert.NewAdhocProvider(
				con.CACertificate(),
				con.ServerKey(),
				24*time.Hour,
				5*time.Minute,
				con.Logger(),
			), nil
		},
		nil,
	).(cert.Provider)
}

// CertificatePath returns the name of the directory containing certificates.
func (con *Container) CertificatePath() string {
	return os.Getenv("CERTIFICATE_PATH")
}

// CACertificate returns the CA certificate used to generate ad-hoc certificates.
func (con *Container) CACertificate() *tls.Certificate {
	return con.get(
		"cert.adhoc.ca",
		func() (interface{}, error) {
			certificate, err := tls.LoadX509KeyPair(
				path.Join(con.CertificatePath(), "ca.crt"),
				path.Join(con.CertificatePath(), "ca.key"),
			)
			if err != nil {
				return nil, err
			}

			return &certificate, nil
		},
		nil,
	).(*tls.Certificate)
}

// ServerKey returns the private key used to generate ad-hoc certificates.
func (con *Container) ServerKey() *rsa.PrivateKey {
	return con.get(
		"cert.adhoc.server-key",
		func() (interface{}, error) {
			raw, err := ioutil.ReadFile(
				path.Join(con.CertificatePath(), "server.key"),
			)
			if err != nil {
				return nil, err
			}

			block, unused := pem.Decode(raw)
			if len(unused) > 0 {
				return nil, err
			}

			return x509.ParsePKCS1PrivateKey(block.Bytes)
		},
		nil,
	).(*rsa.PrivateKey)
}
