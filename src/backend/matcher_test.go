package backend_test

import (
	"strings"

	"github.com/icecave/honeycomb/src/backend"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matcher", func() {
	Describe("NewMatcher", func() {
		It("accepts a pattern with no wildcards", func() {
			subject, err := backend.NewMatcher("host.domain-name.tld")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("host.domain-name.tld"))
		})

		It("accepts a pattern with a wildcard prefix", func() {
			subject, err := backend.NewMatcher("*.domain-name.tld")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("*.domain-name.tld"))
		})

		It("accepts a pattern with a wildcard suffix", func() {
			subject, err := backend.NewMatcher("host.*")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("host.*"))
		})

		It("accepts a pattern with a wildcard prefix and suffix", func() {
			subject, err := backend.NewMatcher("*.host.*")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("*.host.*"))
		})

		It("accepts a wildcard pattern with no domain part", func() {
			subject, err := backend.NewMatcher("*.*")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("*.*"))
		})

		It("accepts a catch-all wildcard pattern", func() {
			subject, err := backend.NewMatcher("*")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(subject).ShouldNot(BeNil())

			Expect(subject.Pattern).To(Equal("*"))
		})

		DescribeTable(
			"it rejects patterns with invalid domain names",
			func(pattern string) {
				subject, err := backend.NewMatcher(pattern)
				Expect(err).To(MatchError("'" + pattern + "' is not a valid domain pattern"))
				Expect(subject).Should(BeNil())
			},
			Entry("invalid character", "/"),
			Entry("dot before hyphen", "foo.-bar"),
			Entry("hypen before dot", "foo.-bar"),
			Entry("dot before dot", "foo..bar"),
			Entry("leading hyphen", "-foo"),
			Entry("leading dot", ".foo"),
			Entry("trailing hyphen", "foo-"),
			Entry("trailing dot", "foo."),
			Entry("first atom too long", strings.Repeat("x", 64)+".bar"),
			Entry("last atom too long", "foo."+strings.Repeat("x", 64)),
			Entry("only atom too long", strings.Repeat("x", 64)),
		)
	})

})
