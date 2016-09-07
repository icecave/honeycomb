package generator

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

func newTemplateCertificate(
	serverName name.ServerName,
	notBeforeOffset time.Duration,
	notAfterOffset time.Duration,
) (*x509.Certificate, error) {
	serialNumber, err := rand.Int(
		rand.Reader,
		new(big.Int).Lsh(big.NewInt(1), 128),
	)
	if err != nil {
		return nil, err
	}

	if notBeforeOffset == 0 {
		notBeforeOffset = DefaultNotBeforeOffset
	}

	if notAfterOffset == 0 {
		notAfterOffset = DefaultNotAfterOffset
	}

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: serverName.Unicode},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{serverName.Punycode},
		NotBefore:             time.Now().Add(notBeforeOffset),
		NotAfter:              time.Now().Add(notAfterOffset),
	}, nil
}
