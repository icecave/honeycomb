package backend_test

import (
	"github.com/icecave/honeycomb/src/backend"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Endpoint", func() {
	DescribeTable(
		"returns the correct scheme",
		func(isTLS, isWebSocket bool, expected string) {
			endpoint := &backend.Endpoint{IsTLS: isTLS}
			result := endpoint.GetScheme(isWebSocket)
			Expect(result).To(Equal(expected))
		},
		Entry("Non-TLS + HTTP", false, false, "http"),
		Entry("TLS + HTTP", true, false, "https"),
		Entry("Non-TLS + WebSocket", false, true, "ws"),
		Entry("TLS + WebSocket", true, true, "wss"),
	)
})
