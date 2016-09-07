package request_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/icecave/honeycomb/src/request"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResponseWriter", func() {
	var (
		inner       *fakeResponseWriter
		transaction *request.Transaction
		subject     *request.Writer
	)

	BeforeEach(func() {
		inner = &fakeResponseWriter{
			header: http.Header{"X-Header": []string{"<value>"}},
		}
		transaction = &request.Transaction{}
		subject = &request.Writer{
			Inner:       inner,
			Transaction: transaction,
		}
	})

	Describe("Header", func() {
		It("returns the headers from the inner writer", func() {
			Expect(subject.Header()).To(Equal(inner.header))
		})

		It("returns nil if the writer is closed", func() {
			subject.Inner = nil
			Expect(subject.Header()).To(BeNil())
		})
	})

	Describe("Write", func() {
		It("increments the byte counter", func() {
			subject.Write([]byte("<buffer>"))
			Expect(transaction.BytesOut).To(Equal(8))
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

		It("does nothing if the writer is closed", func() {
			subject.Inner = nil
			subject.Write([]byte("<buffer>"))
			Expect(transaction.BytesOut).To(Equal(0))
			Expect(inner.buffer).To(BeNil())
		})
	})

	Describe("WriteHeader", func() {
		It("writes the headers on the inner writer", func() {
			subject.WriteHeader(http.StatusNotFound)
			Expect(inner.statusCode).To(Equal(http.StatusNotFound))
		})

		It("transitions the transaction to the 'responded' state", func() {
			subject.WriteHeader(http.StatusNotFound)
			Expect(transaction.State).To(Equal(request.StateResponded))
		})

		It("does nothing if the writer is closed", func() {
			subject.Inner = nil
			subject.WriteHeader(http.StatusNotFound)
			Expect(inner.statusCode).To(Equal(0))
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
			Expect(err).To(MatchError("The inner response writer does not implement http.Hijacker."))
		})

		It("fails if the writer is closed", func() {
			subject.Inner = nil
			_, _, err := subject.Hijack()
			Expect(err).To(MatchError("The response writer is closed."))
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
