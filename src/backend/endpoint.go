package backend

// Endpoint holds information about a back-end HTTP(s) server.
type Endpoint struct {
	// A human readable description of what the end-point is, not necessarily
	// unique to this endpoint.
	Description string

	// Address hosts the network address of the back-end server, including the
	// port number or name.
	Address string

	// IsTLS indicates whether or not the back-end server is expecting a TLS
	// connection. If true, the "https://" or "wss://" scheme is used; otherwise,
	// "http://" or "ws://" is used.
	IsTLS bool
}
