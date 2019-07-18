package health_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/icecave/honeycomb/src/docker/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPChecker", func() {
	var (
		server          *httptest.Server
		serverURL       *url.URL
		subject         *health.HTTPChecker
		slowServer      *httptest.Server
		slowServerURL   *url.URL
		slowSubject     *health.HTTPChecker
		responseCode    int
		responseMessage string
	)

	handler := func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(responseCode)
		io.WriteString(response, responseMessage)
	}

	slowHandler := func(response http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Second)
		response.WriteHeader(responseCode)
		io.WriteString(response, responseMessage)
	}

	BeforeEach(func() {
		server = httptest.NewUnstartedServer(http.HandlerFunc(handler))
		server.StartTLS()

		slowServer = httptest.NewUnstartedServer(http.HandlerFunc(slowHandler))
		slowServer.StartTLS()

		serverURL, _ = url.Parse(server.URL)
		slowServerURL, _ = url.Parse(slowServer.URL)

		subject = &health.HTTPChecker{
			Address: serverURL.Host,
		}

		slowSubject = &health.HTTPChecker{
			Address: slowServerURL.Host,
			Client: &http.Client{
				Timeout: 500 * time.Millisecond,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		}
	})

	AfterEach(func() {
		server.Close()
		slowServer.Close()
	})

	DescribeTable(
		"Check",
		func(code int, message string, expected health.Status) {
			responseCode = code
			responseMessage = message
			Expect(subject.Check()).To(Equal(expected))
		},
		Entry(
			"healthy response",
			http.StatusOK,
			"<ok message>",
			health.Status{IsHealthy: true, Message: "<ok message>"},
		),
		Entry(
			"unhealthy response",
			http.StatusServiceUnavailable,
			"<error message>",
			health.Status{IsHealthy: false, Message: "<error message>"},
		),
	)

	Describe("Check", func() {
		It("defaults to localhost", func() {
			_, port, _ := net.SplitHostPort(serverURL.Host)
			subject.Address = fmt.Sprintf(":%s", port)

			responseCode = http.StatusOK
			responseMessage = "<message>"

			expected := health.Status{
				IsHealthy: true,
				Message:   "<message>",
			}

			Expect(subject.Check()).To(Equal(expected))
		})

		It("returns an unhealthy status when the address is invalid", func() {
			subject.Address = "x"

			expected := health.Status{
				IsHealthy: false,
				Message:   "address x: missing port in address",
			}

			Expect(subject.Check()).To(Equal(expected))
		})

		It("returns an unhealthy status when the check is too slow", func() {
			expected := health.Status{
				IsHealthy: false,
				Message:   fmt.Sprintf("Get %s/.honeycomb/health-check: net/http: request canceled (Client.Timeout exceeded while awaiting headers)", slowServerURL),
			}

			Expect(slowSubject.Check()).To(Equal(expected))
		})

		It("returns an unhealthy status when the server is unreachable", func() {
			server.Close()

			result := subject.Check()

			Expect(result.IsHealthy).To(BeFalse())
			Expect(result.Message).To(ContainSubstring("connection refused"))
		})
	})
})
