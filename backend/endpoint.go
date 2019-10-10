package backend

// Endpoint holds information about a back-end HTTP(s) server.
type Endpoint struct {
	// A human readable description of what the end-point is, not necessarily
	// unique to this endpoint.
	Description string

	// Address holds the network address of the back-end server, including the
	// port number or name.
	Address string

	// TLSMode indicates whether or not the back-end server is expecting a TLS
	// connection.
	TLSMode TLSMode
}

// TLSMode is an enumerationo of the TLS "modes" used by an endpoint.
type TLSMode int

const (
	// TLSDisabled indicates that the endpoint does not use TLS.
	TLSDisabled TLSMode = iota

	// TLSEnabled indicates that the endpoint does use TLS.
	TLSEnabled

	// TLSInsecure indicates that the endpoint does use TLS, but that its
	// certificate details should not be verified.
	TLSInsecure
)
