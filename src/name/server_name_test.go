package name_test

import (
	"strings"

	"github.com/icecave/honeycomb/src/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerName", func() {
	Describe("ParseServerName", func() {
		It("accepts valid international domains", func() {
			result := name.ParseServerName("host.dømåin-name.tld")
			Expect(result).To(Equal(name.ServerName{
				Unicode:  "host.dømåin-name.tld",
				Punycode: "host.xn--dmin-name-62a1s.tld",
			}))
		})

		It("normalizes the name", func() {
			result := name.ParseServerName("HOST.DØMÅIN-NAME.TLD")
			Expect(result).To(Equal(name.ServerName{
				Unicode:  "host.dømåin-name.tld",
				Punycode: "host.xn--dmin-name-62a1s.tld",
			}))
		})

		DescribeTable(
			"it rejects patterns with invalid server names",
			func(serverName string) {
				defer func() {
					err := recover()
					Expect(err).Should(HaveOccurred())
				}()
				name.ParseServerName(serverName)
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

	Describe("TryParseServerName", func() {
		It("accepts valid international domains", func() {
			result, err := name.TryParseServerName("host.dømåin-name.tld")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).To(Equal(name.ServerName{
				Unicode:  "host.dømåin-name.tld",
				Punycode: "host.xn--dmin-name-62a1s.tld",
			}))
		})

		It("normalizes the name", func() {
			result, err := name.TryParseServerName("HOST.DØMÅIN-NAME.TLD")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).To(Equal(name.ServerName{
				Unicode:  "host.dømåin-name.tld",
				Punycode: "host.xn--dmin-name-62a1s.tld",
			}))
		})

		DescribeTable(
			"it rejects patterns with invalid server names",
			func(serverName string) {
				_, err := name.TryParseServerName(serverName)
				Expect(err).To(HaveOccurred())
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
			Entry("too long for IDNA encoding", strings.Repeat("x", 65536)+"\uff00"),
		)
	})
})
