package request

// Handler is a HTTP request handler that operates on a request context, rather
// than separate writer / request objects.
type Handler interface {
	Serve(*Context)
}
