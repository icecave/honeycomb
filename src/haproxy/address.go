package haproxy

import (
	"fmt"
	"net"

	proxyproto "github.com/pires/go-proxyproto"
)

// NewAddr creates an `Addr` with supplied network, host and port as strings.
func NewAddr(network, host, port string) Addr {
	return Addr{
		net:  network,
		addr: net.JoinHostPort(host, port),
	}
}

// NewProxyAddr creates an `Addr` struct from supplied `proxyproto.AddressFamilyAndProtocol`, `net.IP` and port as `unit16`.
func NewProxyAddr(proto proxyproto.AddressFamilyAndProtocol, addr net.IP, port uint16) Addr {
	return Addr{
		net:  convertProxyProtocolToString(proto),
		addr: net.JoinHostPort(addr.String(), fmt.Sprintf("%d", port)),
	}
}

// Addr is a `net.Addr` compatible struct for use with `net.Conn` `RemoteAddr()` and `LocalAddr()`
type Addr struct {
	net  string
	addr string
}

// Network returns the name of the network (for example, "tcp", "udp")
func (a Addr) Network() string {
	return a.net
}

// String returns the string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (a Addr) String() string {
	return a.addr
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
