package backend

import (
	"context"
	"sync"

	"github.com/icecave/honeycomb/src/name"
)

// Cache is a Locator that caches the results of another locator.
type Cache struct {
	Next Locator

	m     sync.RWMutex
	cache map[name.ServerName]cacheEntry
}

type cacheEntry struct {
	Endpoint *Endpoint
	Score    int
}

// Locate finds the back-end HTTP server for the given server name.
//
// It returns a score indicating the strength of the match. A value of 0 or
// less indicates that no match was made, in which case ep is nil.
//
// A non-zero score can be returned with a nil endpoint, indicating that the
// request should not be routed.
func (c *Cache) Locate(ctx context.Context, serverName name.ServerName) (ep *Endpoint, score int) {
	c.m.RLock()
	e, ok := c.cache[serverName]
	c.m.RUnlock()

	if ok {
		return e.Endpoint, e.Score
	}

	ep, score = c.Next.Locate(ctx, serverName)

	c.m.Lock()
	defer c.m.Unlock()

	if c.cache == nil {
		c.cache = map[name.ServerName]cacheEntry{}
	}

	c.cache[serverName] = cacheEntry{ep, score}

	return ep, score
}

// Clear clears the cache.
func (c *Cache) Clear() {
	c.m.Lock()
	defer c.m.Unlock()

	c.cache = nil
}
