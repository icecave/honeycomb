package loader

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"path"
)

// FileLoader loads certificates and keys from the local file system.
type FileLoader struct {
	BasePath string
}

// LoadCertificate reads an x509 certificate from a file in PEM format and
// returns the parsed certificate.
func (loader *FileLoader) LoadCertificate(_ context.Context, certFile string) (*x509.Certificate, error) {
	bytes, err := loader.readPEM(certFile)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(bytes)
}

// LoadPrivateKey reads an RSA certificate from a file in PEM formt and
// returns the parsed key.
func (loader *FileLoader) LoadPrivateKey(_ context.Context, keyFile string) (*rsa.PrivateKey, error) {
	bytes, err := loader.readPEM(keyFile)
	if err != nil {
		return nil, err
	}

	return x509.ParsePKCS1PrivateKey(bytes)
}

func (loader *FileLoader) readPEM(file string) ([]byte, error) {
	p := path.Join(loader.BasePath, file)
	raw, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	block, unused := pem.Decode(raw)
	if len(unused) > 0 {
		return nil, errors.New("unused data in PEM file")
	}

	return block.Bytes, nil
}
