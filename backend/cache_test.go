package backend_test

import (
	"context"

	"github.com/icecave/honeycomb/backend"
	"github.com/icecave/honeycomb/name"
	"github.com/icecave/honeycomb/static"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache", func() {
	var (
		next    static.Locator
		subject *backend.Cache
	)

	BeforeEach(func() {
		next = static.Locator{}.
			With("foo", &backend.Endpoint{Address: "static-foo:443"})

		subject = &backend.Cache{
			Next: next,
		}
	})

	Describe("Locate", func() {
		It("locates endpoints from the inner locator", func() {
			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("static-foo:443"))
			Expect(score).To(BeNumerically(">", 0))
		})

		It("returns prior matches from the cache", func() {
			// prime the cache
			subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)

			subject.Next = static.Locator{}

			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("static-foo:443"))
			Expect(score).To(BeNumerically(">", 0))
		})

		It("returns nil and a non-positive score if none of the inner locators can locate the endpoint", func() {
			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("unknown"),
			)
			Expect(endpoint).To(BeNil())
			Expect(score).To(BeNumerically("<=", 0))
		})
	})

	Describe("Clear", func() {
		It("invalidates the cache", func() {
			// prime the cache
			subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)

			subject.Next = static.Locator{}
			subject.Clear()

			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)
			Expect(endpoint).Should(BeNil())
			Expect(score).To(BeNumerically("<=", 0))
		})
	})

})
