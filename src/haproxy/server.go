package haproxy

import (
	"net"
)

// Listener is a struct for providing a `net.Listener` compatible struct that checks for PROXY headers on new connections.
type Listener struct {
	l net.Listener
}

// NewListener returns a `Listener` wrapping a supplied `net.Listener`
func NewListener(l net.Listener) net.Listener {
	return &Listener{
		l: l,
	}
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.l.Accept()
	if err != nil {
		return c, err
	}

	pc, err := NewConn(c)
	return pc, err
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *Listener) Close() error {
	return l.l.Close()
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.l.Addr()
}
