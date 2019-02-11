package haproxy

import (
	"bufio"
	"net"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

// Conn is a `net.Conn` compatible struct that handles PROXY header checking.
type Conn struct {
	rd  *bufio.Reader
	c   net.Conn
	r   net.Addr
	l   net.Addr
	hdr *proxyproto.Header
}

// NewConn wraps a connection with a `bufio.Reader` for checking if the
// PROXY protocol headers are present
func NewConn(nc net.Conn) (net.Conn, error) {
	c := &Conn{
		c:  nc,
		rd: bufio.NewReader(nc),
	}
	if err := c.ProxyInit(); err != nil {
		return nil, err
	}
	return c, nil
}

// ProxyInit checks for the PROXY protocol headers, and if not returns
// the connection unmolested.
func (c *Conn) ProxyInit() error {
	pc, err := proxyproto.Read(c.rd)
	if err != nil &&
		err != proxyproto.ErrNoProxyProtocol &&
		err != proxyproto.ErrInvalidLength {
		return err
	} else if err == proxyproto.ErrInvalidLength {
		// Failed to parse header, the connection is probably about to drop anyway,
		// it's not a PROXY header at least.
		return nil
	}

	c.hdr = pc
	return nil
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	return c.rd.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.c.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.c.Close()
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	if c.hdr == nil {
		return c.c.LocalAddr()
	}
	if c.l == nil {
		c.l = NewProxyAddr(c.hdr.TransportProtocol, c.hdr.DestinationAddress, c.hdr.DestinationPort)
	}
	return c.l
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	if c.hdr == nil {
		return c.c.RemoteAddr()
	}
	if c.r == nil {
		c.r = NewProxyAddr(c.hdr.TransportProtocol, c.hdr.SourceAddress, c.hdr.SourcePort)
	}
	return c.r
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}
