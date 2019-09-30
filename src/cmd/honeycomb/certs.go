package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"path"

	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/frontend/cert/generator"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// certificateResolver returns the cert.Resolver used to obtain certificates for
// TLS requests.
func certificateResolver(
	config *cmd.Config,
	serverKey *rsa.PrivateKey,
	logger *log.Logger,
) *cert.Resolver {
	r := &cert.Resolver{}

	if p, ok := fileCertificateProvider(config, logger); ok {
		r.Recognized = append(r.Recognized, p)
		r.Unrecognized = append(r.Unrecognized, p)
	}

	if p, ok := acmeCertificateProvider(config, logger); ok {
		r.Recognized = append(r.Recognized, p)
	}

	if p, ok := adhocCertificateProvider(config, serverKey, logger); ok {
		r.Recognized = append(r.Recognized, p)
		r.Unrecognized = append(r.Unrecognized, p)
	}

	return r
}

// fileCertificateProvider returns a cert provider that loads certificate/key
// pairs from disk.
func fileCertificateProvider(
	config *cmd.Config,
	logger *log.Logger,
) (cert.Provider, bool) {
	i, err := os.Stat(config.Certificates.BasePath)

	if os.IsNotExist(err) {
		logger.Printf(
			"File certificate provider is DISABLED, %s does not exist",
			config.Certificates.BasePath,
		)

		return nil, false
	}

	if err != nil {
		logger.Printf(
			"File certificate provider is DISABLED, unable to stat %s: %s",
			config.Certificates.BasePath,
			err,
		)

		return nil, false
	}

	if !i.IsDir() {
		logger.Printf(
			"File certificate provider is DISABLED, %s is not a directory",
			config.Certificates.BasePath,
		)

		return nil, false
	}

	logger.Printf(
		"File certificate provider is ENABLED, loading certificates from %s",
		config.Certificates.BasePath,
	)

	return &cert.FileProvider{
		BasePath: config.Certificates.BasePath,
		Logger:   logger,
	}, true
}

// acmeCertificateProvider returns a cert provider that obtains certificates
// from an ACME server.
func acmeCertificateProvider(
	config *cmd.Config,
	logger *log.Logger,
) (cert.Provider, bool) {
	if config.Certificates.ACME.Email == "" {
		logger.Printf(
			"ACME certificate provider is DISABLED, no email address was configured",
		)
		return nil, false
	}

	logger.Printf(
		"ACME certificate provider is ENABLED, acquiring certificates as %s",
		config.Certificates.ACME.Email,
	)

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	if config.Certificates.ACME.CachePath != "" {
		logger.Printf(
			"ACME certificate provider is caching certificates in %s",
			config.Certificates.ACME.CachePath,
		)

		m.Cache = autocert.DirCache(config.Certificates.ACME.CachePath)
	}

	if config.Certificates.ACME.URL != "" {
		logger.Printf(
			"ACME certificate provider is using a non-default endpoint of %s",
			config.Certificates.ACME.URL,
		)

		m.Client = &acme.Client{
			DirectoryURL: config.Certificates.ACME.URL,
		}
	}

	return &cert.ACMEProvider{
		Manager: m,
		Logger:  logger,
	}, true
}

// adhocCertificateProvider returns a cert provider that generates certificates
// on the fly.
func adhocCertificateProvider(
	config *cmd.Config,
	serverKey *rsa.PrivateKey,
	logger *log.Logger,
) (cert.Provider, bool) {
	issuer, err := tls.LoadX509KeyPair(
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerCertificate),
		path.Join(config.Certificates.BasePath, config.Certificates.IssuerKey),
	)
	if err != nil {
		logger.Printf(
			"Adhoc certificate provider is DISABLED, unable to load issuer certificate: %s",
			err,
		)
		return nil, false
	}

	x509Cert, err := x509.ParseCertificate(issuer.Certificate[0])
	if err != nil {
		logger.Printf(
			"Adhoc certificate provider is DISABLED, unable to parse issuer certificate: %s",
			err,
		)
		return nil, false
	}

	issuer.Leaf = x509Cert

	logger.Printf(
		"Adhoc certificate provider is ENABLED, issuing certificates as '%s'",
		issuer.Leaf.Issuer.CommonName,
	)

	return &cert.AdhocProvider{
		Generator: &generator.IssuerSignedGenerator{
			IssuerCertificate: issuer.Leaf,
			IssuerKey:         issuer.PrivateKey.(*rsa.PrivateKey),
			ServerKey:         serverKey,
		},
		Logger: logger,
	}, true
}
