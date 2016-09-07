package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
)

// LoadX509Certificate reads an x509 certificate from a file in PEM format and
// returns the parsed certificate.
func LoadX509Certificate(path string) (*x509.Certificate, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, unused := pem.Decode(raw)
	if len(unused) > 0 {
		return nil, errors.New("unused data in PEM file")
	}

	return x509.ParseCertificate(block.Bytes)
}

// LoadPrivateKey reads an RSA certificate from a file in PEM formt and returns
// the parsed key.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, unused := pem.Decode(raw)
	if len(unused) > 0 {
		return nil, errors.New("unused data in PEM file")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
