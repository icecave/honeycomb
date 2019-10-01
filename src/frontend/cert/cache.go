package cert

import (
	"log"
	"sync"
	"time"

	"github.com/icecave/honeycomb/src/name"
)

// Cache is an in-memory cache of results obtained from providers.
type Cache struct {
	Logger *log.Logger

	m       sync.RWMutex
	entries map[string]*cacheEntry
}

// Get retrieves an entry from the cache.
//
// If there is an existing entry for n with a rank that is worse (numerically
// larger) than r, it is discarded.
func (c *Cache) Get(n name.ServerName, r int) (ProviderResult, bool) {
	c.m.RLock()
	e, ok := c.entries[n.Punycode]
	c.m.RUnlock()

	if !ok {
		return ProviderResult{}, false
	}

	// If the entry exists but it's not ranked highly enough, ignore it.
	if e.Rank > r {
		return ProviderResult{}, false
	}

	// If the entry is is still valid, use it.
	if e.Provider.IsValid(e.Result) {
		return e.Result, true
	}

	// Otherwise, we'll try to remove it.
	c.m.Lock()
	defer c.m.Unlock()

	x, ok := c.entries[n.Punycode]

	if ok {
		// If there is an entry in the cache, but it's not the invalid one,
		// it must have been replaced by another goroutine.
		if x != e {
			return x.Result, true
		}

		// Otherwise, we delete the stale entry.
		delete(c.entries, n.Punycode)

		c.Logger.Printf(
			"Certificate for '%s', expires at %s (%s), issued by '%s' has been invalidated and removed from the cache.",
			e.Result.Certificate.Leaf.Subject.CommonName,
			e.Result.Certificate.Leaf.NotAfter.Format(time.RFC3339),
			time.Until(e.Result.Certificate.Leaf.NotAfter),
			e.Result.Certificate.Leaf.Issuer.CommonName,
		)
	}

	return ProviderResult{}, false
}

// Put stores an entry in the cache.
//
// If there is an existing entry for n with a rank better (numerically less)
// than r, it is retained. Otherwise, it is replaced with pr.
//
// Whichever result is retained in the cache is returned. The boolean return
// value is true if pr is stored, or false if some existing entry was retained.
func (c *Cache) Put(
	n name.ServerName,
	r int,
	p Provider,
	pr ProviderResult,
) (ProviderResult, bool) {
	if pr.ExcludeFromCache {
		panic("can not store result, it is marked as uncachable")
	}

	c.m.Lock()
	defer c.m.Unlock()

	if c.entries == nil {
		c.entries = map[string]*cacheEntry{}
	}

	// If there is an existing, higher-ranked entry, do not replace it.
	if x, ok := c.entries[n.Punycode]; ok && x.Rank < r {
		return x.Result, false
	}

	c.entries[n.Punycode] = &cacheEntry{
		Rank:     r,
		Provider: p,
		Result:   pr,
	}

	c.Logger.Printf(
		"Certificate for '%s', expires at %s (%s), issued by '%s' has been added to the cache.",
		pr.Certificate.Leaf.Subject.CommonName,
		pr.Certificate.Leaf.NotAfter.Format(time.RFC3339),
		time.Until(pr.Certificate.Leaf.NotAfter),
		pr.Certificate.Leaf.Issuer.CommonName,
	)

	return pr, true
}

// cacheEntry is an entry in the certificate cache.
type cacheEntry struct {
	// Rank is a ranking of the desirability of the cached certificate.
	Rank int

	// Provider is the provider that provided the cached certificate.
	Provider Provider

	// Result is the cached provider result.
	Result ProviderResult
}
