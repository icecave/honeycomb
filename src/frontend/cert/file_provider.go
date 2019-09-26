package cert

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

const certExtension = ".crt"
const keyExtension = ".key"

// FileProvider is an implementation of Provider that loads certificates from
// files on disk.
type FileProvider struct {
	// BasePath is the directory containing the certificate and key files.
	BasePath string

	// Logger is the destination for messages loaded and ignored certificates.
	Logger *log.Logger
}

// GetCertificate loads the certificate used for the given server name from
// disk, if available.
//
// It searches for appropriate key/cert pairs in files located under p.BasePath.
// Certificate files are associated with a server name by naming them after the
// punycode representation of the server name.
//
// If there is no exact match for a server name, each subdomain is searched.
// This is to allow for the use of wildcard certificates or certificates with
// multiple SANs.
//
// For example, the server name "www.en.example.org" will result in a search of
// the following file names, in order:
//
//   - www.en.example.org.crt
//   - _.en.example.org.crt
//   - _.example.org.crt
//   - _.org.crt
//   - _.crt
func (p *FileProvider) GetCertificate(
	n name.ServerName,
	_ *tls.ClientHelloInfo,
) (ProviderResult, bool) {
	for _, stem := range filenameStems(n) {
		c, ok, err := p.load(n, stem)

		if err != nil {
			p.Logger.Printf(
				"Certificate '%s' ignored for '%s', %s",
				stem+certExtension,
				n.Unicode,
				err,
			)

			continue
		}

		if ok {
			p.Logger.Printf(
				"Loaded certificate for '%s' from '%s', expires at %s, issued by '%s'",
				n.Unicode,
				stem+certExtension,
				c.Leaf.NotAfter.Format(time.RFC3339),
				c.Leaf.Issuer.CommonName,
			)

			return ProviderResult{
				Certificate: c,
			}, true
		}
	}

	return ProviderResult{}, false
}

// IsValid returns true if the given provider result should still be
// considered valid.
//
// The behavior is undefined if the result was not obtained from this
// provider.
//
// This implementation always returns true, as there's generally no reason to reload the same
// file from disk. It's possible that the file on disk has been replaced,
// but since this would typically be provided by a Docker secret, the
// service would be restarted in that case anyway.
//
// The cached result will still be discarded when the certificate expires.
func (p *FileProvider) IsValid(ProviderResult) bool {
	return true
}

// load attempts to load a certificate for n at a file with the given stem.
func (p *FileProvider) load(
	n name.ServerName,
	stem string,
) (*tls.Certificate, bool, error) {
	base := path.Join(p.BasePath, stem)
	certFile := base + certExtension
	keyFile := base + keyExtension

	if _, err := os.Stat(certFile); err != nil {
		// skip this stem if no such file exists
		if os.IsNotExist(err) {
			return nil, false, nil
		}

		// there was some more fundamental filesystem problem
		return nil, false, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, false, err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, false, err
	}

	cert.Leaf = x509Cert

	// only use this certificate if it's actually valid for this server name
	if err := cert.Leaf.VerifyHostname(n.Punycode); err != nil {
		return nil, false, err
	}

	return &cert, true, nil
}

// filenameStems returns a list of filenames (without extensions) that
// should be searched for a certificate for the given server name, in order of
// preference.
func filenameStems(n name.ServerName) []string {
	tail := n.Punycode
	stems := []string{n.Punycode} // always look for an exact match

	for {
		parts := strings.SplitN(tail, ".", 2)
		if len(parts) == 1 {
			return stems
		}

		tail = parts[1]
		stems = append(stems, "_."+tail, tail)
	}
}
