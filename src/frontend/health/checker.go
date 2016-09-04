package health

// Checker is an interface for querying the health of the HTTPS server.
type Checker interface {
	// Check returns information about the health of the HTTPS server.
	Check() Status
}
