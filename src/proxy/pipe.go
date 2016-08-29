package proxy

import (
	"io"
	"net"
	"sync"
)

// Pipe sets up a bidirectional pipe between two connections.
func Pipe(lhs, rhs net.Conn) error {
	var group sync.WaitGroup
	results := make(chan error, 2)

	group.Add(2)
	go pipe(results, lhs, rhs)
	go pipe(results, rhs, lhs)
	group.Wait()

	return <-results
}

func pipe(results chan error, source io.Reader, target io.Writer) {
	_, err := io.Copy(target, source)
	results <- err
}
