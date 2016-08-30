package backend

import "sync"

// StaticLocator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by looking up back-ends in a static list.
type StaticLocator struct {
	endpoints map[string]*Endpoint
	mutex     sync.RWMutex
}

// Locate finds the back-end HTTP server for the given domain name.
func (locator *StaticLocator) Locate(domainName string) *Endpoint {
	locator.mutex.RLock()
	endpoint := locator.endpoints[domainName]
	locator.mutex.RUnlock()

	return endpoint
}

// CanLocate checks if the given domain name can be resolved to a back-end.
func (locator *StaticLocator) CanLocate(domainName string) bool {
	locator.mutex.RLock()
	_, ok := locator.endpoints[domainName]
	locator.mutex.RUnlock()

	return ok
}

// Add creates a new mapping from domain name to back-end HTTP server.
func (locator *StaticLocator) Add(domainName string, endpoint *Endpoint) {
	locator.mutex.Lock()
	if locator.endpoints == nil {
		locator.endpoints = map[string]*Endpoint{}
	}
	locator.endpoints[domainName] = endpoint
	locator.mutex.Unlock()
}
