package docker

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/icecave/honeycomb/src/backend"
)

// DefaultPollInterval is the default interval between rebuilds of the service
// list.
const DefaultPollInterval = 30 * time.Second

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by querying a Docker swarm manager for services.
type Locator struct {
	PollInterval time.Duration
	Loader       *ServiceLoader
	Logger       *log.Logger

	done     chan struct{}
	services atomic.Value // []ServiceInfo
}

// NewLocator returns a new Docker locator.
func NewLocator(
	pollInterval time.Duration,
	loader *ServiceLoader,
	logger *log.Logger,
) *Locator {
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}

	return &Locator{
		PollInterval: pollInterval,
		Loader:       loader,
		Logger:       logger,
		done:         make(chan struct{}),
	}
}

// Locate finds the back-end HTTP server for the given server name.
func (locator *Locator) Locate(ctx context.Context, serverName string) *backend.Endpoint {
	if services, ok := locator.services.Load().([]ServiceInfo); ok {
		for _, info := range services {
			if info.Matcher.Match(serverName) {
				return info.Endpoint
			}
		}
	}

	return nil
}

// Run polls Docker for service information until Stop() is called.
func (locator *Locator) Run() {
	services := locator.load()
	locator.diff(nil, services)

	for {
		select {
		case <-time.After(locator.PollInterval):
			s := locator.load()
			locator.diff(services, s)
			services = s
		case <-locator.done:
			return
		}
	}
}

// Stop shuts down the locator and cleans up any resources used.
func (locator *Locator) Stop() {
	close(locator.done)
}

func (locator *Locator) load() []ServiceInfo {
	new, err := locator.Loader.Load(context.Background())

	if err == nil {
		locator.services.Store(new)
	} else {
		locator.Logger.Printf("docker: %s", err)
	}

	return new
}

func (locator *Locator) diff(old []ServiceInfo, new []ServiceInfo) {
	for _, info := range old {
		log := true
		for _, other := range new {
			if info.Equal(other) {
				log = false
				break
			}
		}

		if log {
			locator.Logger.Printf(
				"docker: Removed route from '%s' to '%s' (%s)",
				info.Matcher.Pattern,
				info.Name,
				info.Endpoint.Description,
			)
		}
	}

	for _, info := range new {
		log := true
		for _, other := range old {
			if info.Equal(other) {
				log = false
				break
			}
		}

		if log {
			locator.Logger.Printf(
				"docker: Added route from '%s' to '%s' (%s)",
				info.Matcher.Pattern,
				info.Name,
				info.Endpoint.Description,
			)
		}
	}
}
