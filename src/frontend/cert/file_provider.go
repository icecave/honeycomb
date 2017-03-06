package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

const certExtension = ".crt"
const keyExtension = ".key"

// FileProvider a certificate provider that reads certificates from a loader.
type FileProvider struct {
	BasePath string
	Logger   *log.Logger

	mutex sync.RWMutex
	cache map[string]*tls.Certificate
}

// GetCertificate attempts to fetch an existing certificate for the given
// server name. If no such certificate exists, it generates one.
func (p *FileProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	cert, err := p.GetExistingCertificate(ctx, n)
	if err != nil {
		return nil, err
	} else if cert != nil {
		return cert, err
	}

	return nil, errors.New("file provider can not generated certificates")
}

// GetExistingCertificate attempts to fetch an existing certificate for the
// given server name. It never generates new certificates. A non-nil error
// indicates an error with the provider itself; otherwise, a nil certificate
// indicates a failure to find an existing certificate.
func (p *FileProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	if cert, ok := p.findInCache(n); ok {
		return cert, nil
	}

	for _, filename := range p.resolveFilenames(n) {
		cert, err := p.loadCertificate(ctx, n, filename)
		if cert != nil || err != nil {
			return cert, err
		}
	}

	return nil, nil
}

func (p *FileProvider) loadCertificate(
	ctx context.Context,
	n name.ServerName,
	filename string,
) (*tls.Certificate, error) {
	base := path.Join(p.BasePath, filename)
	certFile := base + certExtension
	keyFile := base + keyExtension

	if _, err := os.Stat(certFile); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}

	cert.Leaf = x509Cert

	err = cert.Leaf.VerifyHostname(n.Punycode)
	if err != nil {
		if p.Logger != nil {
			p.Logger.Printf(
				"Certificate '%s' ignored for '%s', %s",
				filename+certExtension,
				n.Unicode,
				err,
			)
		}

		return nil, nil
	}

	if p.Logger != nil {
		p.Logger.Printf(
			"Loaded certificate for '%s' from '%s', expires at %s, issued by '%s'",
			n.Unicode,
			filename+certExtension,
			cert.Leaf.NotAfter.Format(time.RFC3339),
			cert.Leaf.Issuer.CommonName,
		)
	}

	p.writeToCache(n, &cert)

	return &cert, nil
}

func (p *FileProvider) resolveFilenames(
	n name.ServerName,
) (filenames []string) {
	tail := n.Punycode
	filenames = []string{tail}

	for {
		parts := strings.SplitN(tail, ".", 2)
		if len(parts) == 1 {
			return
		}

		tail = parts[1]
		filenames = append(filenames, "_."+tail, tail)
	}
}

func (p *FileProvider) findInCache(
	n name.ServerName,
) (*tls.Certificate, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	cert, ok := p.cache[n.Unicode]

	return cert, ok
}

func (p *FileProvider) writeToCache(
	n name.ServerName,
	cert *tls.Certificate,
) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.cache == nil {
		p.cache = map[string]*tls.Certificate{}
	}

	p.cache[n.Unicode] = cert
}
