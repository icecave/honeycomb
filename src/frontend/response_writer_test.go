package frontend_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/icecave/honeycomb/src/frontend"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResponseWriter", func() {
	var inner *fakeResponseWriter
	var subject *frontend.ResponseWriter

	BeforeEach(func() {
		inner = &fakeResponseWriter{
			header: http.Header{"X-Header": []string{"<value>"}},
		}
		subject = &frontend.ResponseWriter{Inner: inner}
	})

	Describe("Header", func() {
		It("returns the headers from the inner writer", func() {
			Expect(subject.Header()).To(Equal(inner.header))
		})
	})

	Describe("Write", func() {
		It("increments the byte counter", func() {
			subject.Write([]byte("<buffer>"))
			Expect(subject.Size).To(Equal(8))
		})

		It("writes the buffer to the inner writer", func() {
			buffer := []byte("<buffer>")
			subject.Write(buffer)
			Expect(inner.buffer).To(Equal(buffer))
		})

		It("returns the result of the inner writer", func() {
			Expect(subject.Write([]byte("<buffer>"))).To(Equal(8))
		})

		It("writes the headers if they haven't yet been written", func() {
			subject.Write([]byte("<buffer>"))
			Expect(inner.statusCode).To(Equal(http.StatusOK))
		})

		It("does not write the headers if they've already been written", func() {
			subject.WriteHeader(http.StatusNotFound)
			subject.Write([]byte("<buffer>"))
			Expect(inner.statusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("WriteHeader", func() {
		It("writes the headers on the inner writer", func() {
			subject.WriteHeader(http.StatusNotFound)
			Expect(inner.statusCode).To(Equal(http.StatusNotFound))
		})

		It("calls FirstWrite with the status code", func() {
			statusCode := 0
			subject.FirstWrite = func(sc int) {
				statusCode = sc
			}

			subject.WriteHeader(http.StatusNotFound)
			Expect(statusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("Flush", func() {
		It("flushes the inner writer if it implements http.Flusher", func() {
			flusher := fakeFlusher{}
			subject.Inner = &flusher
			subject.Flush()
			Expect(flusher.flushed).To(BeTrue())
		})

		It("does nothing if the inner writer does not implement http.Flusher", func() {
			subject.Flush()
		})
	})

	Describe("Hijack", func() {
		It("hijacks the inner writer if it implements http.Hijacker", func() {
			hijacker := fakeHijacker{}
			subject.Inner = &hijacker
			subject.Hijack()
			Expect(hijacker.hijacked).To(BeTrue())
		})

		It("returns the result of the inner writer", func() {
			hijacker := fakeHijacker{}
			subject.Inner = &hijacker
			_, _, err := subject.Hijack()
			Expect(err).To(MatchError("<result>"))
		})

		It("fails if the inner writer does not implement http.Hijacker", func() {
			_, _, err := subject.Hijack()
			Expect(err).To(MatchError("The wrapped response does not implement http.Hijacker."))
		})
	})

	Describe("CloseNotifier", func() {
		It("returns the result of the inner writer", func() {
			closeNotifier := fakeCloseNotifier{}
			subject.Inner = &closeNotifier
			Expect(subject.CloseNotify()).To(BeIdenticalTo(closeNotifier.channel))
		})
	})
})

type fakeResponseWriter struct {
	header     http.Header
	buffer     []byte
	statusCode int
}

func (writer *fakeResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *fakeResponseWriter) Write(data []byte) (int, error) {
	writer.buffer = append(writer.buffer, data...)

	return len(data), nil
}

func (writer *fakeResponseWriter) WriteHeader(statusCode int) {
	writer.statusCode = statusCode
}

type fakeFlusher struct {
	fakeResponseWriter
	flushed bool
}

func (flusher *fakeFlusher) Flush() {
	flusher.flushed = true
}

type fakeHijacker struct {
	fakeResponseWriter
	hijacked bool
}

func (hijacker *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker.hijacked = true
	return nil, nil, fmt.Errorf("<result>")
}

type fakeCloseNotifier struct {
	fakeResponseWriter
	channel <-chan bool
}

func (notifier *fakeCloseNotifier) CloseNotify() <-chan bool {
	if notifier.channel == nil {
		notifier.channel = make(chan bool)
	}

	return notifier.channel
}
