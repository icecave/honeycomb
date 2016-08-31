package backend

import "context"

// AggregateLocator combines multiple locators to find endpoints.
type AggregateLocator []Locator

// Locate finds the back-end HTTP server for the given server name.
func (locator AggregateLocator) Locate(ctx context.Context, serverName string) *Endpoint {
	for _, loc := range locator {
		if endpoint := loc.Locate(ctx, serverName); endpoint != nil {
			return endpoint
		}
	}

	return nil
}

// CanLocate checks if the given server name can be resolved to a back-end.
func (locator AggregateLocator) CanLocate(ctx context.Context, serverName string) bool {
	for _, loc := range locator {
		if loc.CanLocate(ctx, serverName) {
			return true
		}
	}

	return false
}
