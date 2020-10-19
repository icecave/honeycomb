package cert_test

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/icecave/honeycomb/frontend/cert"
	"github.com/icecave/honeycomb/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var (
	multiTLSCertificatePrimary = &tls.Certificate{
		Certificate: [][]byte{
			[]byte("imacert"),
		},
	}
	multiTLSCertificateSecondary = &tls.Certificate{
		Certificate: [][]byte{
			[]byte("imanothercert"),
		},
	}
)

type multiTestType int

const (
	primaryCertReturn multiTestType = iota
	secondaryCertReturn
	errorReturn
	nilReturn
)

var errCertError = errors.New("error certificate")

type mockPrimaryProvider struct{}

func (m *mockPrimaryProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	if n.Unicode == "primary.cert" {
		return multiTLSCertificatePrimary, nil
	}

	if n.Unicode == "primary.error" {
		return nil, errCertError
	}

	return nil, nil
}

func (m *mockPrimaryProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	if n.Unicode == "primary.cert" {
		return multiTLSCertificatePrimary, nil
	}

	if n.Unicode == "primary.error" {
		return nil, errCertError
	}

	return nil, nil
}

type mockSecondaryProvider struct{}

func (m *mockSecondaryProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	if n.Unicode == "secondary.cert" {
		return multiTLSCertificateSecondary, nil
	}

	if n.Unicode == "secondary.error" {
		return nil, errCertError
	}

	return nil, nil
}

func (m *mockSecondaryProvider) GetExistingCertificate(
	ctx context.Context,
	n name.ServerName,
) (*tls.Certificate, error) {
	if n.Unicode == "secondary.cert" {
		return multiTLSCertificateSecondary, nil
	}

	if n.Unicode == "secondary.error" {
		return nil, errCertError
	}

	return nil, nil
}

var multiTestList = []TableEntry{
	Entry(
		"returns the primary provider certificate",
		name.Parse("primary.cert"),
		primaryCertReturn,
	),
	Entry(
		"returns the secondary provider certificate",
		name.Parse("secondary.cert"),
		secondaryCertReturn,
	),
	Entry(
		"returns an error from the primary provider",
		name.Parse("primary.error"),
		errorReturn,
	),
	Entry(
		"returns an error from the secondary provider",
		name.Parse("secondary.error"),
		errorReturn,
	),
	Entry(
		"returns no error or certificate",
		name.Parse("somedomain.com"),
		nilReturn,
	),
}

var _ = Describe("MultiProvider", func() {
	DescribeTable(
		"GetCertificate",
		func(n name.ServerName, tt multiTestType) {
			p := &cert.MultiProvider{
				PrimaryProvider:   &mockPrimaryProvider{},
				SecondaryProvider: &mockSecondaryProvider{},
			}
			c, err := p.GetCertificate(context.Background(), n)
			switch tt {
			case primaryCertReturn:
				Expect(c).To(BeEquivalentTo(multiTLSCertificatePrimary))
				Expect(err).NotTo(HaveOccurred())
			case secondaryCertReturn:
				Expect(c).To(BeEquivalentTo(multiTLSCertificateSecondary))
				Expect(err).NotTo(HaveOccurred())
			case errorReturn:
				Expect(err).To(HaveOccurred())
			case nilReturn:
				Expect(err).NotTo(HaveOccurred())
			}
		},
		multiTestList...,
	)
	DescribeTable(
		"GetExistingCertificate",
		func(n name.ServerName, tt multiTestType) {
			p := &cert.MultiProvider{
				PrimaryProvider:   &mockPrimaryProvider{},
				SecondaryProvider: &mockSecondaryProvider{},
			}
			c, err := p.GetExistingCertificate(context.Background(), n)
			switch tt {
			case primaryCertReturn:
				Expect(c).To(BeEquivalentTo(multiTLSCertificatePrimary))
				Expect(err).NotTo(HaveOccurred())
			case secondaryCertReturn:
				Expect(c).To(BeEquivalentTo(multiTLSCertificateSecondary))
				Expect(err).NotTo(HaveOccurred())
			case errorReturn:
				Expect(err).To(HaveOccurred())
			case nilReturn:
				Expect(err).NotTo(HaveOccurred())
			}
		},
		multiTestList...,
	)
})
