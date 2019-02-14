package proxyprotocol

import (
	"net"
	"strings"

	proxyproto "github.com/pires/go-proxyproto"
)

// NewProxyAddr creates an Addr struct from supplied
// proxyproto.AddressFamilyAndProtocol, net.IP and port as uint16.
func NewProxyAddr(proto proxyproto.AddressFamilyAndProtocol, addr net.IP, port uint16) net.Addr {
	network := convertProxyProtocolToString(proto)
	switch strings.ToLower(network) {
	case "unix", "unixstream", "unixdgram":
		return &net.UnixAddr{
			Net:  network,
			Name: addr.String(),
		}
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{
			IP:   addr,
			Port: int(port),
		}
	default:
		return &net.TCPAddr{
			IP:   addr,
			Port: int(port),
		}
	}
}

func convertProxyProtocolToString(afp proxyproto.AddressFamilyAndProtocol) string {
	if afp.IsIPv4() {
		// IPv4
		if afp.IsStream() {
			return "tcp4"
		}
		return "udp4"
	} else if afp.IsIPv6() {
		// IPv6
		if afp.IsStream() {
			return "tcp6"
		}
		return "udp6"
	} else if afp.IsUnix() {
		// UnixSocket
		if afp.IsStream() {
			return "unix"
		}
		return "unixgram"
	} else if afp.IsUnspec() {
		// Unspecified
		return "unspec"
	}
	return "unspec"
}
