package docker

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/engine-api/types/swarm"
	"github.com/icecave/honeycomb/src/backend"
)

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by querying a Docker swarm manager for services.
type Locator struct {
	// The Docker client used to query the available services.
	client client.APIClient

	// The service inspector used to extract endpoint information.
	inspector *inspector

	// Logger specifies a logger for messages related to locating backends.
	logger *log.Logger

	// The certificate cache, maps domain name to TLS certificate.
	cache atomic.Value

	// A mutex for ensuring only one goroutine rebuilds the cache concurrently.
	mutex TryMutex
}

// NewLocator returns a new Docker locator.
func NewLocator(
	client client.APIClient,
	logger *log.Logger,
) *Locator {
	return &Locator{
		client:    client,
		inspector: &inspector{client},
		logger:    logger,
		mutex:     NewTryMutex(),
	}
}

// Locate finds the back-end HTTP server for the given domain name.
func (locator *Locator) Locate(ctx context.Context, domainName string) *backend.Endpoint {
	endpoint := locator.find(domainName)

	if endpoint == nil {
		locator.rebuild(ctx)
		endpoint = locator.find(domainName)
	}

	return endpoint
}

// CanLocate checks if the given domain name can be resolved to a back-end.
func (locator *Locator) CanLocate(ctx context.Context, domainName string) bool {
	return locator.Locate(ctx, domainName) != nil
}

func (locator *Locator) find(domainName string) *backend.Endpoint {
	if cache := locator.cache.Load(); cache != nil {
		for _, item := range cache.([]*cacheItem) {
			if item.matcher(domainName) {
				return item.endpoint
			}
		}
	}

	return nil
}

func (locator *Locator) newItem(
	ctx context.Context,
	name string,
	svc *swarm.Service,
) (*cacheItem, error) {
	matcher, pattern, err := locator.inspector.DomainMatcher(ctx, svc)
	if err != nil {
		return nil, err
	}

	port, err := locator.inspector.Port(ctx, svc)
	if err != nil {
		return nil, err
	}

	isTLS, err := locator.inspector.IsTLS(ctx, port, svc)
	if err != nil {
		return nil, err
	}

	return &cacheItem{
		matcher,
		pattern,
		&backend.Endpoint{
			Name: name,
			Address: net.JoinHostPort(
				svc.Spec.Name,
				strconv.Itoa(port),
			),
			IsTLS: isTLS,
		},
	}, nil
}

func (locator *Locator) rebuild(ctx context.Context) {
	if !locator.mutex.TryLockOrWaitWithContext(ctx) {
		return
	}
	defer locator.mutex.Unlock()

	filter := filters.NewArgs()
	filter.Add("label", domainLabel)
	services, err := locator.client.ServiceList(
		ctx,
		types.ServiceListOptions{Filter: filter},
	)
	if err != nil {
		locator.logger.Printf("docker: %s", err)
		return
	}

	var cache []*cacheItem

	for _, svc := range services {
		name := fmt.Sprintf(
			"%s@%s",
			svc.Spec.TaskTemplate.ContainerSpec.Image,
			svc.Spec.Name,
		)
		item, err := locator.newItem(ctx, name, &svc)

		if err == nil {
			locator.logger.Printf(
				"docker: Discovered '%s', domains matching '%s' will be fowarded to %s (tls: %t)",
				name,
				item.pattern,
				item.endpoint.Address,
				item.endpoint.IsTLS,
			)
			item.endpoint.Name = name
			cache = append(cache, item)
		} else {
			locator.logger.Printf(
				"docker: Discovered '%s', but it is unusable: %s",
				name,
				err,
			)
		}
	}

	locator.cache.Store(cache)
}

type cacheItem struct {
	matcher  domainMatcher
	pattern  string
	endpoint *backend.Endpoint
}
