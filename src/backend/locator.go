package backend

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI).
type Locator interface {
	// Locate finds the back-end HTTP server for the given domain name.
	Locate(domainName string) (*Endpoint, bool)

	// CanLocate checks if the given domain name can be resolved to a back-end.
	CanLocate(domainName string) bool
}
