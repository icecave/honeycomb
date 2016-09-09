package health

// Checker is an interface for querying the health of the server.
type Checker interface {
	// Check returns the health-check status.
	Check() Status
}
