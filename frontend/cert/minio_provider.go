package cert

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/icecave/honeycomb/name"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioProvider a certificate provider that reads certificates from a loader.
type MinioProvider struct {
	Client     *minio.Client
	BucketName string
	CacheAge   time.Duration
	Logger     *log.Logger

	mutex sync.RWMutex
	cache map[string]*minioCacheItem
}

type minioCacheItem struct {
	Certificate *tls.Certificate
	LastSeen    time.Time
}

// errMinioCertNotFound is returned when the certificate is not found or not correctly formed in minio.
var errMinioCertNotFound = errors.New("certificate not found")

// NewMinioProvider returns a MinioProvider preconfigured and setup for use as a Provider.
func NewMinioProvider(
	logger *log.Logger,
	endpoint, region, bucketName, accessKeyID, secretAccessKey string,
	useSSL bool,
) (Provider, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, err
	}

	if ok, err := c.BucketExists(context.Background(), bucketName); err == nil && !ok {
		err := c.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{
			Region: region,
		})
		if err != nil {
			return nil, err
		}
	}

	return &MinioProvider{
		Client:     c,
		BucketName: bucketName,
		Logger:     logger,
	}, err
}

// GetCertificate attempts to fetch an existing certificate for the given
// server name. If no such certificate exists, it generates one.
func (p *MinioProvider) GetCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	cert, err := p.GetExistingCertificate(ctx, n)
	if err != nil {
		return nil, err
	} else if cert != nil {
		return cert, err
	}

	return nil, fmt.Errorf("minio %w", ErrProviderGenerateUnsupported)
}

// GetExistingCertificate attempts to fetch an existing certificate for the
// given server name. It never generates new certificates. A non-nil error
// indicates an error with the provider itself; otherwise, a nil certificate
// indicates a failure to find an existing certificate.
func (p *MinioProvider) GetExistingCertificate(ctx context.Context, n name.ServerName) (*tls.Certificate, error) {
	// If cache has not expired, attempt to find in cache.
	if !p.expiredInCache(n) {
		if cert, ok := p.findInCache(n); ok {
			return cert, nil
		}
	}

	for _, objectName := range p.resolveObjectNames(n) {
		// No cache (or expired), attempt to look up in redis
		// but if redis is down or broken suddenly, we should reuse the
		// cached certificate until it's replaced.
		if cert, err := p.getMinioCertificate(ctx, objectName); err == nil {
			p.writeToCache(n, cert)

			return cert, nil
		}
	}

	// fail through to getting it from the cache.
	if cert, ok := p.findInCache(n); ok {
		p.Logger.Printf("expired but falling through to cache for %s", n.Unicode)

		return cert, nil
	}

	// and finally we just fail.
	return nil, nil
}

func (p *MinioProvider) getMinioObject(ctx context.Context, objectName string) (*minio.Object, error) {
	return p.Client.GetObject(
		ctx,
		p.BucketName,
		fmt.Sprintf("%s.crt", objectName),
		minio.GetObjectOptions{},
	)
}

func (p *MinioProvider) getMinioCertificate(ctx context.Context, objectName string) (*tls.Certificate, error) {
	var (
		certObj, keyObj *minio.Object
		err             error
	)

	if certObj, err = p.getMinioObject(ctx, fmt.Sprintf("%s.crt", objectName)); err != nil {
		return nil, err
	}

	if keyObj, err = p.getMinioObject(ctx, fmt.Sprintf("%s.key", objectName)); err != nil {
		return nil, err
	}

	certBuf := bytes.NewBuffer(nil)
	if _, err = io.Copy(certBuf, certObj); err != nil {
		return nil, err
	}

	keyBuf := bytes.NewBuffer(nil)
	if _, err = io.Copy(keyBuf, keyObj); err != nil {
		return nil, err
	}

	if cert, err := tls.X509KeyPair(certBuf.Bytes(), keyBuf.Bytes()); err == nil {
		return &cert, nil
	}

	return nil, errMinioCertNotFound
}

func (p *MinioProvider) resolveObjectNames(
	n name.ServerName,
) (filenames []string) {
	tail := n.Punycode
	filenames = []string{tail}

	for {
		parts := strings.SplitN(tail, ".", 2)
		if len(parts) == 1 {
			return
		}

		tail = parts[1]
		filenames = append(filenames, "_."+tail, tail)
	}
}

func (p *MinioProvider) expiredInCache(n name.ServerName) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if item, ok := p.cache[n.Unicode]; ok {
		if p.CacheAge > 0 && item.LastSeen.Before(time.Now().Add(-1*p.CacheAge)) {
			return true
		}
	}

	return false
}

func (p *MinioProvider) findInCache(
	n name.ServerName,
) (*tls.Certificate, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if item, ok := p.cache[n.Unicode]; ok {
		return item.Certificate, ok
	}

	return nil, false
}

func (p *MinioProvider) writeToCache(
	n name.ServerName,
	cert *tls.Certificate,
) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.cache == nil {
		p.cache = map[string]*minioCacheItem{}
	}

	item := &minioCacheItem{
		Certificate: cert,
		LastSeen:    time.Now(),
	}

	p.cache[n.Unicode] = item
}
