package proxyprotocol_test

import (
	"fmt"
	"net"

	"github.com/icecave/honeycomb/proxyprotocol"
	proxyproto "github.com/pires/go-proxyproto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PROXY Protocol", func() {
	Describe("Connection", func() {
		It("accepts PROXY v2 connections", func() {
			server, client := net.Pipe()

			go func() {
				header := &proxyproto.Header{
					Command:            proxyproto.PROXY,
					DestinationAddress: net.ParseIP("127.0.0.1"),
					DestinationPort:    12345,
					SourceAddress:      net.ParseIP("127.127.127.127"),
					SourcePort:         31337,
					TransportProtocol:  proxyproto.TCPv4,
					Version:            2,
				}
				n, err := header.WriteTo(client)
				Expect(n).To(BeNumerically(">", 0))
				Expect(err).NotTo(HaveOccurred())
				err = client.Close()
				Expect(err).NotTo(HaveOccurred())
			}()

			pServer, err := proxyprotocol.NewConn(server)
			Expect(err).ShouldNot(HaveOccurred())
			defer pServer.Close()
			Expect(pServer.RemoteAddr().String()).To(Equal("127.127.127.127:31337"))
			Expect(pServer.LocalAddr().String()).To(Equal("127.0.0.1:12345"))
		})

		It("accepts PROXY v1 connections", func() {
			server, client := net.Pipe()

			go func() {
				header := &proxyproto.Header{
					Command:            proxyproto.PROXY,
					DestinationAddress: net.ParseIP("127.0.0.1"),
					DestinationPort:    12345,
					SourceAddress:      net.ParseIP("127.127.127.127"),
					SourcePort:         31337,
					TransportProtocol:  proxyproto.TCPv4,
					Version:            1,
				}
				n, err := header.WriteTo(client)
				Expect(n).To(BeNumerically(">", 0))
				Expect(err).NotTo(HaveOccurred())
				err = client.Close()
				Expect(err).NotTo(HaveOccurred())
			}()

			pServer, err := proxyprotocol.NewConn(server)
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

			pServer, err := proxyprotocol.NewConn(server)
			Expect(err).ShouldNot(HaveOccurred())
			defer pServer.Close()
			Expect(pServer.RemoteAddr().String()).To(Equal("pipe"))
			Expect(pServer.LocalAddr().String()).To(Equal("pipe"))
		})

	})

})
