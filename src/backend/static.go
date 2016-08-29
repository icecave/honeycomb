package backend

// StaticLocator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by looking up back-ends in a static list.
type StaticLocator struct {
	endpoints map[string]*Endpoint
}

// NewStaticLocator creates a new StaticLocator.
func NewStaticLocator() *StaticLocator {
	return &StaticLocator{map[string]*Endpoint{}}
}

// Locate finds the back-end HTTP server for the given domain name.
func (locator *StaticLocator) Locate(domainName string) (*Endpoint, bool) {
	endpoint, ok := locator.endpoints[domainName]
	return endpoint, ok
}

// CanLocate checks if the given domain name can be resolved to a back-end.
func (locator *StaticLocator) CanLocate(domainName string) bool {
	return locator.endpoints[domainName] != nil
}

// Add creates a new mapping from domain name to back-end HTTP server.
func (locator *StaticLocator) Add(domainName string, endpoint *Endpoint) {
	locator.endpoints[domainName] = endpoint
}
