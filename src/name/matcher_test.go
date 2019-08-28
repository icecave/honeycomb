package name_test

import (
	"strings"

	"github.com/icecave/honeycomb/src/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matcher", func() {
	Describe("NewMatcher", func() {
		DescribeTable(
			"accepts valid patterns",
			func(pattern string) {
				subject, err := name.NewMatcher(pattern)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(subject).ShouldNot(BeNil())
				Expect(subject.Pattern).To(Equal(pattern))
			},
			Entry("exact match", "host.dømåin-name.tld"),
			Entry("wildcard prefix", "*.dømåin-name.tld"),
			Entry("wildcard suffix", "host.*"),
			Entry("wildcard", "*.dømåin-name.*"),
			Entry("catch all with dot", "*.*"),
			Entry("catch all", "*"),
		)

		DescribeTable(
			"it rejects patterns with invalid server names",
			func(pattern string) {
				subject, err := name.NewMatcher(pattern)
				Expect(err).To(MatchError("'" + pattern + "' is not a valid server name pattern"))
				Expect(subject).Should(BeNil())
			},
			Entry("empty", ""),
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

	Describe("Match", func() {
		DescribeTable(
			"it returns a positive score when passed a matching server name",
			func(pattern, serverName string) {
				subject, _ := name.NewMatcher(pattern)
				score := subject.Match(name.Parse(serverName))
				Expect(score).To(BeNumerically(">", 0))
			},
			Entry("exact match", "host.dømåin-name.tld", "host.dømåin-name.tld"),
			Entry("wildcard prefix", "*.dømåin-name.tld", "host.dømåin-name.tld"),
			Entry("wildcard suffix", "host.*", "host.dømåin-name.tld"),
			Entry("wildcard", "*.dømåin-name.*", "host.dømåin-name.tld"),
			Entry("catch all with dot", "*.*", "host.dømåin-name.tld"),
			Entry("catch all", "*", "host.dømåin-name.tld"),
		)

		DescribeTable(
			"it returns a non-positive score when passed a non-matching server name",
			func(pattern, serverName string) {
				subject, _ := name.NewMatcher(pattern)
				score := subject.Match(name.Parse(serverName))
				Expect(score).To(BeNumerically("<=", 0))
			},
			Entry("exact match", "host.dømåin-name.tld", "host.different.tld"),
			Entry("wildcard prefix", "*.dømåin-name.tld", "host.different.tld"),
			Entry("wildcard suffix", "host.*", "different.dømåin-name.tld"),
			Entry("wildcard", "*.dømåin-name.*", "host.different.tld"),
			Entry("catch all with dot", "*.*", "no-dot"),
		)

		It("scores wildcard matches appropriately", func() {
			serverName := name.Parse("w.prefix.example.x")

			// note: subject3 is an exact matcher for the server name, but as
			// the same "fixedPart" length as the wildcard subject4, yet it must
			// still score higher.
			subject1, _ := name.NewMatcher("*")
			subject2, _ := name.NewMatcher("*.example.*")
			subject3, _ := name.NewMatcher("*.prefix.example.*")
			subject4, _ := name.NewMatcher("w.prefix.example.x")

			score1 := subject1.Match(serverName)
			Expect(score1).To(BeNumerically(">", 0))

			score2 := subject2.Match(serverName)
			Expect(score2).To(BeNumerically(">", 0))

			score3 := subject3.Match(serverName)
			Expect(score3).To(BeNumerically(">", 0))

			score4 := subject4.Match(serverName)
			Expect(score4).To(BeNumerically(">", 0))

			Expect(score1).To(BeNumerically("<", score2))
			Expect(score2).To(BeNumerically("<", score3))
			Expect(score3).To(BeNumerically("<", score4))
		})
	})
})
