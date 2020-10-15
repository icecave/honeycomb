package cert

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/icecave/honeycomb/name"
)

// RedisProvider a certificate provider that reads certificates from a loader.
type RedisProvider struct {
	Logger   *log.Logger
	Client   *redis.Client
	CacheAge time.Duration

	mutex sync.RWMutex
	cache map[string]*redisCacheItem
}

type redisCacheItem struct {
	Certificate *tls.Certificate
	LastSeen    time.Time
}

// GetCertificate attempts to fetch an existing certificate for the given
// server name. If no such certificate exists, it generates one.
func (p *RedisProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	cert, err := p.GetExistingCertificate(ctx, n)
	if err != nil {
		return nil, err
	} else if cert != nil {
		return cert, err
	}

	return nil, errors.New("redis provider can not generate certificates")
}

func certificateRedisKey(key string) (o string) {
	return fmt.Sprintf("ssl:%s", key)
}

func certAndKeyFromMap(m map[string]string) (cert string, key string, ok bool) {
	if cert, ok = m["certificate"]; !ok {
		return
	}

	key, ok = m["key"]

	return
}

// GetExistingCertificate attempts to fetch an existing certificate for the
// given server name. It never generates new certificates. A non-nil error
// indicates an error with the provider itself; otherwise, a nil certificate
// indicates a failure to find an existing certificate.
func (p *RedisProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	// If cache has not expired, attempt to find in cache.
	if !p.expiredInCache(n) {
		if cert, ok := p.findInCache(n); ok {
			return cert, nil
		}
	}

	// No cache (or expired), attempt to look up in redis
	// but if redis is down or broken suddenly, we should reuse the
	// cached certificate until it's replaced.
	if cert, err := p.getRedisCertificate(ctx, n); err == nil {
		return cert, nil
	}

	// fail through to getting it from the cache.
	if cert, ok := p.findInCache(n); ok {
		p.Logger.Printf("expired but falling through to cache for %s", n.Unicode)
		return cert, nil
	}

	// and finally we just fail.
	return nil, nil
}

func (p *RedisProvider) getRedisCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	r, err := p.Client.HGetAll(ctx, certificateRedisKey(n.Unicode)).Result()
	if err != nil {
		// p.Logger.Printf("failed to retrieve certificate for %s", certificateRedisKey(n.Unicode))
		// p.Logger.Printf("redis error: %s", err)
		return nil, err
	}

	if cr, ck, ok := certAndKeyFromMap(r); ok {
		if cert, err := tls.X509KeyPair([]byte(cr), []byte(ck)); err == nil {
			p.writeToCache(n, &cert)
			return &cert, nil
		}
	}

	return nil, errors.New("certificate not found")
}

// // FlushOldCacheItems will flush any entries from the cache that are older than the specified duration.
// func (p *RedisProvider) FlushOldCacheItems(
// 	d time.Duration,
// ) {
// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	n := time.Now().Add(-1 * d)

// 	for k, v := range p.cache {
// 		if v.LastSeen.Before(n) {
// 			delete(p.cache, k)
// 		}
// 	}
// }

func (p *RedisProvider) expiredInCache(n name.ServerName) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if item, ok := p.cache[n.Unicode]; ok {
		if p.CacheAge > 0 && item.LastSeen.Before(time.Now().Add(-1*p.CacheAge)) {
			return true
		}
	}

	return false
}

func (p *RedisProvider) findInCache(
	n name.ServerName,
) (*tls.Certificate, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if item, ok := p.cache[n.Unicode]; ok {
		return item.Certificate, ok
	}

	return nil, false
}

func (p *RedisProvider) deleteFromCache(
	n name.ServerName,
) (*tls.Certificate, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	item, ok := p.cache[n.Unicode]

	return item.Certificate, ok
}

func (p *RedisProvider) writeToCache(
	n name.ServerName,
	cert *tls.Certificate,
) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.cache == nil {
		p.cache = map[string]*redisCacheItem{}
	}

	item := &redisCacheItem{
		Certificate: cert,
		LastSeen:    time.Now(),
	}

	p.cache[n.Unicode] = item
}
