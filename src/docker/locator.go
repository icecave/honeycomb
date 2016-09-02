package docker

import (
	"context"
	"sync/atomic"

	"github.com/icecave/honeycomb/src/backend"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by querying a Docker swarm manager for services.
type Locator struct {
	Loader *ServiceLoader

	cache atomic.Value
	mutex TryMutex
}

// NewLocator returns a new Docker locator.
func NewLocator(loader *ServiceLoader) *Locator {
	return &Locator{
		Loader: loader,
		mutex:  NewTryMutex(),
	}
}

// Locate finds the back-end HTTP server for the given server name.
func (locator *Locator) Locate(ctx context.Context, serverName string) *backend.Endpoint {
	if info, ok := locator.match(serverName); ok {
		return info.Endpoint
	}

	locator.discover(ctx)

	if info, ok := locator.match(serverName); ok {
		return info.Endpoint
	}

	return nil
}

func (locator *Locator) match(serverName string) (*ServiceInfo, bool) {
	if cache := locator.cache.Load(); cache != nil {
		for _, info := range cache.([]ServiceInfo) {
			if info.Matcher.Match(serverName) {
				return &info, true
			}
		}
	}

	return nil, false
}

func (locator *Locator) discover(ctx context.Context) {
	if !locator.mutex.TryLockOrWaitWithContext(ctx) {
		return
	}

	defer locator.mutex.Unlock()

	cache, err := locator.Loader.Load(ctx)

	if err == nil {
		locator.cache.Store(cache)
	}
}
