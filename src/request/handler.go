package request

// Handler is a HTTP request handler that operates on a transaction, rather
// than separate writer / request objects.
type Handler interface {
	Serve(*Transaction)
}
