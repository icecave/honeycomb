package health_test

import (
	"github.com/icecave/honeycomb/docker/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Status", func() {
	Describe("String", func() {
		It("indicates that the status check passed", func() {
			subject := health.Status{IsHealthy: true, Message: "<message>."}
			Expect(subject.String()).To(Equal("Health-check passed: <message>."))
		})

		It("indicates that the status check failed", func() {
			subject := health.Status{IsHealthy: false, Message: "<message>."}
			Expect(subject.String()).To(Equal("Health-check failed: <message>."))
		})
	})
})
