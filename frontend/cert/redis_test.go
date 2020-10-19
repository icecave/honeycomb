package cert_test

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/icecave/honeycomb/frontend/cert"
	"github.com/icecave/honeycomb/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	realDomainCert = `-----BEGIN CERTIFICATE-----
MIIB5TCCAYygAwIBAgIUSM0nqePsmshbPMgGcbIK95ZpmlMwCgYIKoZIzj0EAwIw
JTEjMCEGA1UEAxMaVGVzdCBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwIBcNMjAxMDE0
MjMyMjAwWhgPMjA1MDEwMDcyMzIyMDBaMBkxFzAVBgNVBAMTDnJlYWxkb21haW4u
Y29tMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEItKG2eC7dVixw5zSawF1QqVs
brg4wIrHzmiXKFCdnjxoFjOHggRZbIbWIy99UBlQdbOuer/vi3o/L36lr9GVR6OB
ozCBoDAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDAYDVR0T
AQH/BAIwADAdBgNVHQ4EFgQUfI8FgjDCNCNe0OAUuNUUpwokz+QwHwYDVR0jBBgw
FoAUzY2yWZMf8753i4leyda2J/72fkkwKwYDVR0RBCQwIoIQKi5yZWFsZG9tYWlu
LmNvbYIOcmVhbGRvbWFpbi5jb20wCgYIKoZIzj0EAwIDRwAwRAIgTSfZshnFyw/O
wegH71O3n4FTJTWQDYHPZdv45Kd8H6cCIFFMUW/6N/V9wdimo/Bmg57lwclYA99u
YSBkYy+xPoyF
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBjzCCATagAwIBAgIUCyGNg36qzRxN2jUZae7LzbaelbEwCgYIKoZIzj0EAwIw
JTEjMCEGA1UEAxMaVGVzdCBDZXJ0aWZpY2F0ZSBBdXRob3JpdHkwIBcNMjAxMDE1
MjMxNzAwWhgPMjA1MDEwMDgyMzE3MDBaMCUxIzAhBgNVBAMTGlRlc3QgQ2VydGlm
aWNhdGUgQXV0aG9yaXR5MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5vAvH4kt
2j6fwke8A6aVYdB0OVV08F9HAU1UxbJhpKa5QsLOQTZmcs1WVl1m7OPkq4Gi7QKP
DpeFvaam8RV/xqNCMEAwDgYDVR0PAQH/BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8w
HQYDVR0OBBYEFM2NslmTH/O+d4uJXsnWtif+9n5JMAoGCCqGSM49BAMCA0cAMEQC
IBUp4kpKBqi9TdUaE4HotSUMx2k+hAqeh/wJS9xWRoeJAiAV4BURvEie4BNFiyKE
oXaYPOJkQqj8flTqUmhLa62pFg==
-----END CERTIFICATE-----`
	realDomainKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIHAt6S1cFRUvCtAA9OZ9M55IRK+AjByzo5tk/qTgR+6ioAoGCCqGSM49
AwEHoUQDQgAEItKG2eC7dVixw5zSawF1QqVsbrg4wIrHzmiXKFCdnjxoFjOHggRZ
bIbWIy99UBlQdbOuer/vi3o/L36lr9GVRw==
-----END EC PRIVATE KEY-----`
)

var mockRedis *miniredis.Miniredis

var _ = Describe("RedisProvider", func() {
	BeforeSuite(func() {
		var err error
		mockRedis, err = miniredis.Run()
		Expect(err).NotTo(HaveOccurred())

		rdb := redis.NewClient(&redis.Options{
			Addr: mockRedis.Addr(),
		})
		defer rdb.Close()
		rdb.HSet(context.Background(), "ssl:realdomain.com", map[string]interface{}{"certificate": realDomainCert, "key": realDomainKey})
	})

	DescribeTable(
		"GetCertificate",
		func(n name.ServerName, expected bool) {
			logger := log.New(os.Stdout, "", log.LstdFlags)
			rdb := redis.NewClient(&redis.Options{
				Addr: mockRedis.Addr(),
			})
			r := &cert.RedisProvider{
				Client:   rdb,
				Logger:   logger,
				CacheAge: time.Second,
			}
			c, err := r.GetCertificate(context.Background(), n)
			if expected {
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.Certificate).To(HaveLen(2))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(c).To(BeNil())
			}
		},
		Entry(
			"returns an existing certificate",
			name.Parse("realdomain.com"),
			true,
		),
		Entry(
			"redis does not generate certificates, returns error",
			name.Parse("notarealdomain.com"),
			false,
		),
		Entry(
			"bad domain name, redis does not generate certificates, returns error",
			name.Parse("√"),
			false,
		),
	)

})
