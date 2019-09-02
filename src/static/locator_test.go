package static_test

import (
	"context"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/static"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Locator", func() {
	var (
		subject static.Locator
	)

	BeforeEach(func() {
		subject = static.Locator{}.
			With("foo", &backend.Endpoint{Address: "foo:443"}).
			With("bar", &backend.Endpoint{Address: "bar1:443"}).
			With("bar", &backend.Endpoint{Address: "bar2:443"})
	})

	Describe("Locate", func() {
		It("matches the endpoints", func() {
			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("foo"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("foo:443"))
			Expect(score).To(BeNumerically(">", 0))
		})

		It("matches the endpoints in order", func() {
			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("bar"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("bar1:443"))
			Expect(score).To(BeNumerically(">", 0))
		})

		It("returns nil and a non-positive score if none of the endpoints match", func() {
			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("unknown"),
			)
			Expect(endpoint).To(BeNil())
			Expect(score).To(BeNumerically("<=", 0))
		})

		It("returns the endpoint with the highest match score", func() {
			subject = static.Locator{}.
				With("*.example.*", &backend.Endpoint{Address: "static1:443"}).
				With("*.prefix.example.*", &backend.Endpoint{Address: "static2:443"})

			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("w.prefix.example.x"),
			)
			Expect(endpoint).ShouldNot(BeNil())
			Expect(endpoint.Address).To(Equal("static2:443"))
			Expect(score).To(BeNumerically(">", 0))
		})
	})

	Describe("With", func() {
		It("panics if the pattern is invalid", func() {
			defer func() {
				err := recover()
				Expect(err).To(HaveOccurred())
			}()
			subject.With("", nil)
		})

		It("allows mapping to a nil endpoint", func() {
			subject = static.Locator{}.
				With("nomatch", nil).
				With("*", &backend.Endpoint{Address: "catch-all:443"})

			endpoint, score := subject.Locate(
				context.Background(),
				name.Parse("nomatch"),
			)
			Expect(endpoint).To(BeNil())
			Expect(score).To(BeNumerically(">", 0))
		})
	})
})
