package backend_test

import (
	"context"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/static"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AggregateLocator", func() {
	var (
		static1, static2 static.Locator
		subject          backend.AggregateLocator
	)

	BeforeEach(func() {
		static1 = static.Locator{}.
			With("foo", &backend.Endpoint{Address: "static1-foo:443"})

		static2 = static.Locator{}.
			With("foo", &backend.Endpoint{Address: "static2-foo:443"}).
			With("bar", &backend.Endpoint{Address: "static2-bar:443"})

		subject = backend.AggregateLocator{static1, static2}
	})

	Describe("Locate", func() {
		It("locates endpoints from the inner locators", func() {
			endpoint := subject.Locate(
				context.Background(),
				name.Parse("bar"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("static2-bar:443"))
		})

		It("searches the inner locators in order", func() {
			endpoint := subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("static1-foo:443"))
		})

		It("returns nil if none of the inner locators can locate the endpoint", func() {
			endpoint := subject.Locate(
				context.Background(),
				name.Parse("unknown"),
			)
			Expect(endpoint).To(BeNil())
		})
	})
})
