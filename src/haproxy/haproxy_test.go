package haproxy_test

import (
	"fmt"
	"net"

	"github.com/icecave/honeycomb/src/haproxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HAProxy", func() {
	Describe("Connection", func() {
		It("accepts PROXY connections", func() {
			server, client := net.Pipe()

			go func() {
				fmt.Fprint(client, "PROXY TCP4 127.127.127.127 127.0.0.1 31337 12345\r\ntest\n")
				err := client.Close()
				Expect(err).NotTo(HaveOccurred())
			}()

			pServer, err := haproxy.NewConn(server)
			Expect(err).ShouldNot(HaveOccurred())
			defer pServer.Close()
			Expect(pServer.RemoteAddr().String()).To(Equal("127.127.127.127:31337"))
			Expect(pServer.LocalAddr().String()).To(Equal("127.0.0.1:12345"))
		})

		It("accepts non-PROXY connections", func() {
			server, client := net.Pipe()

			go func() {
				fmt.Fprint(client, "test\n")
				err := client.Close()
				Expect(err).NotTo(HaveOccurred())
			}()

			pServer, err := haproxy.NewConn(server)
			Expect(err).ShouldNot(HaveOccurred())
			defer pServer.Close()
			Expect(pServer.RemoteAddr().String()).To(Equal("pipe"))
			Expect(pServer.LocalAddr().String()).To(Equal("pipe"))
		})

	})

})
