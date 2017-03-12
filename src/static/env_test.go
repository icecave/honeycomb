package static

import (
	"context"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("FromEnv", func() {
	Describe("fromEnv", func() {
		DescribeTable(
			"it produces the correct route",
			func(env string, expected *backend.Endpoint) {
				locator, err := fromEnv([]string{env})
				Expect(err).ShouldNot(HaveOccurred())

				endpoint := locator.Locate(context.Background(), name.Parse("foo.com"))
				Expect(endpoint).To(Equal(expected))
			},
			Entry("TLS (https)", "ROUTE_FOO=foo.* https://foo.backend.com:1234", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:1234",
				IsTLS:       true,
			}),
			Entry("non-TLS (http)", "ROUTE_FOO=foo.* http://foo.backend.com:1234", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:1234",
				IsTLS:       false,
			}),
			Entry("TLS (wss)", "ROUTE_FOO=foo.* wss://foo.backend.com:1234", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:1234",
				IsTLS:       true,
			}),
			Entry("non-TLS (ws)", "ROUTE_FOO=foo.* ws://foo.backend.com:1234", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:1234",
				IsTLS:       false,
			}),
			Entry("TLS (https, implicit port)", "ROUTE_FOO=foo.* https://foo.backend.com", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:443",
				IsTLS:       true,
			}),
			Entry("non-TLS (http, implicit port)", "ROUTE_FOO=foo.* http://foo.backend.com", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:80",
				IsTLS:       false,
			}),
			Entry("TLS (wss, implicit port)", "ROUTE_FOO=foo.* wss://foo.backend.com", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:443",
				IsTLS:       true,
			}),
			Entry("non-TLS (ws, implicit port)", "ROUTE_FOO=foo.* ws://foo.backend.com", &backend.Endpoint{
				Description: "FOO",
				Address:     "foo.backend.com:80",
				IsTLS:       false,
			}),
			Entry("custom description", "ROUTE_FOO=foo.* https://foo.backend.com:1234 This is the description!", &backend.Endpoint{
				Description: "This is the description!",
				Address:     "foo.backend.com:1234",
				IsTLS:       true,
			}),
		)

		It("allows multiple routes", func() {
			env := []string{
				"ROUTE_FOO=foo.* https://foo.backend.com:1234",
				"ROUTE_BAR=bar.* https://bar.backend.com:1234",
			}

			locator, err := fromEnv(env)

			Expect(err).ShouldNot(HaveOccurred())

			endpoint := locator.Locate(
				context.Background(),
				name.Parse("foo.com"),
			)
			Expect(endpoint.Address).To(Equal("foo.backend.com:1234"))

			endpoint = locator.Locate(
				context.Background(),
				name.Parse("bar.com"),
			)
			Expect(endpoint.Address).To(Equal("bar.backend.com:1234"))
		})

		It("ignores other environment variables", func() {
			env := []string{"PATH=/usr/local/bin"}

			locator, err := fromEnv(env)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(locator).To(HaveLen(0))
		})

		It("returns an error if the match pattern is invalid", func() {
			env := []string{"ROUTE_FOO=/ https://backend"}

			_, err := fromEnv(env)

			Expect(err).Should(HaveOccurred())
		})

		It("returns an error if the URL can not be parsed", func() {
			env := []string{"ROUTE_FOO=www ://backend"}

			_, err := fromEnv(env)

			Expect(err).Should(HaveOccurred())
		})
	})
})
