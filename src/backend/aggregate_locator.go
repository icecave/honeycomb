package backend

import "context"

// AggregateLocator combines multiple locators to find endpoints.
type AggregateLocator []Locator

// Locate finds the back-end HTTP server for the given domain name.
func (locator AggregateLocator) Locate(ctx context.Context, domainName string) *Endpoint {
	for _, loc := range locator {
		if endpoint := loc.Locate(ctx, domainName); endpoint != nil {
			return endpoint
		}
	}

	return nil
}

// CanLocate checks if the given domain name can be resolved to a back-end.
func (locator AggregateLocator) CanLocate(ctx context.Context, domainName string) bool {
	for _, loc := range locator {
		if loc.CanLocate(ctx, domainName) {
			return true
		}
	}

	return false
}
