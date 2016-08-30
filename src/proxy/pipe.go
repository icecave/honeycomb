package proxy

import (
	"io"
	"net"
)

// Pipe sets up a bidirectional pipe between two connections.
func Pipe(lhs, rhs net.Conn) error {
	results := make(chan error, 2)

	go pipe(results, lhs, rhs)
	go pipe(results, rhs, lhs)

	if err := <-results; err != nil {
		return err
	}

	return <-results
}

func pipe(results chan<- error, source io.Reader, target io.Writer) {
	_, err := io.Copy(target, source)
	results <- err
}
