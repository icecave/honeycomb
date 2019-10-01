package cert_test

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"

	. "github.com/icecave/honeycomb/src/frontend/cert"
	"github.com/icecave/honeycomb/src/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type provider struct {
	IsValidFunc func(ProviderResult) bool
}

func (p *provider) GetCertificate(name.ServerName, *tls.ClientHelloInfo) (ProviderResult, bool) {
	panic("not implemented")
}

func (p *provider) IsValid(r ProviderResult) bool {
	if p.IsValidFunc == nil {
		return true
	}

	return p.IsValidFunc(r)
}

var _ = Describe("type Cache", func() {
	var (
		logger     *log.Logger
		cache      *Cache
		prov       *provider
		cert       *tls.Certificate
		serverName name.ServerName
	)

	BeforeEach(func() {
		logger = log.New(ioutil.Discard, "", 0)
		cache = &Cache{
			Logger: logger,
		}
		prov = &provider{}
		cert = &tls.Certificate{
			Leaf: &x509.Certificate{},
		}
		serverName = name.Parse("example.com")
	})

	Describe("func Get()", func() {
		It("returns the result if the matching entry has a good enough rank", func() {
			cache.Put(
				serverName,
				100000,
				prov,
				ProviderResult{
					Certificate: cert,
				},
			)

			r, ok := cache.Get(serverName, 100000)
			Expect(ok).To(BeTrue())
			Expect(r.Certificate).To(Equal(cert))
		})

		It("returns false if there is no matching entry", func() {
			_, ok := cache.Get(serverName, 100000)
			Expect(ok).To(BeFalse())
		})

		It("returns false if the matching entry does not have a good enough rank", func() {
			cache.Put(
				serverName,
				100000,
				prov,
				ProviderResult{
					Certificate: cert,
				},
			)

			_, ok := cache.Get(serverName, 1000)
			Expect(ok).To(BeFalse())
		})

		It("returns false if the entry is invalid", func() {
			prov.IsValidFunc = func(ProviderResult) bool {
				return false
			}

			cache.Put(
				serverName,
				100000,
				prov,
				ProviderResult{
					Certificate: cert,
				},
			)

			_, ok := cache.Get(serverName, 100000)
			Expect(ok).To(BeFalse())
		})

		It("removes invalid entries from the cache", func() {
			prov.IsValidFunc = func(ProviderResult) bool {
				return false
			}

			cache.Put(
				serverName,
				100000,
				prov,
				ProviderResult{
					Certificate: cert,
				},
			)

			_, ok := cache.Get(serverName, 100000)
			Expect(ok).To(BeFalse())

			prov.IsValidFunc = nil

			_, ok = cache.Get(serverName, 100000)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("func Put()", func() {

	})
})
