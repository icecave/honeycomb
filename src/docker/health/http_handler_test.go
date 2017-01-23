package health_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/icecave/honeycomb/src/docker/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPHandler", func() {
	var (
		healthCheckURL = "https://localhost/.honeycomb/health-check"
		subject        *health.HTTPHandler
	)

	BeforeEach(func() {
		subject = &health.HTTPHandler{}
	})

	DescribeTable(
		"CanHandle",
		func(target string, expected bool) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			Expect(subject.CanHandle(request)).To(Equal(expected))
		},
		Entry("health-check URL", healthCheckURL, true),
		Entry("incorrect host", "https://www.domain.tld/.honeycomb/health-check", false),
		Entry("incorrect path", "https://localhost/.honeycomb/elsewhere", false),
	)

	Describe("ServeHTTP", func() {
		Describe("when there is a no checker configured", func() {
			It("writes a healthy response", func() {
				writer := &httptest.ResponseRecorder{Body: &bytes.Buffer{}}
				request := httptest.NewRequest(http.MethodGet, healthCheckURL, nil)
				subject.ServeHTTP(writer, request)

				Expect(writer.Code).To(Equal(http.StatusOK))
				Expect(writer.Body.String()).To(Equal("The server is accepting requests, but no health-checker is configured."))
			})
		})

		DescribeTable(
			"when there is a checker configured",
			func(isHealthy bool, statusCode int) {
				subject.Checker = &fakeChecker{
					health.Status{
						IsHealthy: isHealthy,
						Message:   "<message>",
					},
				}
				writer := &httptest.ResponseRecorder{Body: &bytes.Buffer{}}
				request := httptest.NewRequest(http.MethodGet, "/anything", nil)
				subject.ServeHTTP(writer, request)

				Expect(writer.Code).To(Equal(statusCode))
				Expect(writer.Body.String()).To(Equal("<message>"))
			},
			Entry("writes a healthy response", true, http.StatusOK),
			Entry("writes an unhealthy response", false, http.StatusServiceUnavailable),
		)

		DescribeTable(
			"when there is a logger configured",
			func(isHealthy bool, logOutput string) {
				var buffer bytes.Buffer
				subject.Logger = log.New(&buffer, "", 0)
				subject.Checker = &fakeChecker{
					health.Status{
						IsHealthy: isHealthy,
						Message:   "<message>",
					},
				}
				writer := &httptest.ResponseRecorder{}
				request := httptest.NewRequest(http.MethodGet, "/anything", nil)
				subject.ServeHTTP(writer, request)

				Expect(buffer.String()).To(Equal(logOutput))
			},
			Entry("does not log healthy checks", true, ""),
			Entry("logs unhealthy checks", false, "Health-check failed: <message>\n"),
		)
	})
})

type fakeChecker struct {
	Status health.Status
}

func (checker *fakeChecker) Check() health.Status {
	return checker.Status
}
