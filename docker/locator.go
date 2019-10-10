package docker

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/icecave/honeycomb/backend"
	"github.com/icecave/honeycomb/name"
)

// DefaultPollInterval is the default interval between rebuilds of the service
// list.
const DefaultPollInterval = 30 * time.Second

// Locator finds a back-end HTTP server based on the server name in TLS
// requests (SNI) by querying a Docker swarm manager for services.
type Locator struct {
	PollInterval time.Duration
	Loader       *ServiceLoader
	Cache        *backend.Cache
	Logger       *log.Logger

	done     chan struct{}
	services atomic.Value // []ServiceInfo
}

// Locate finds the back-end HTTP server for the given server name.
//
// It returns a score indicating the strength of the match. A value of 0 or less
// indicates that no match was made, in which case ep is nil.
//
// A non-zero score can be returned with a nil endpoint, indicating that the
// request should not be routed.
func (locator *Locator) Locate(
	ctx context.Context,
	serverName name.ServerName,
) (ep *backend.Endpoint, score int) {
	if services, ok := locator.services.Load().([]ServiceInfo); ok {
		for _, info := range services {
			if s := info.Matcher.Match(serverName); s > score {
				ep = info.Endpoint
				score = s
			}
		}
	}

	return ep, score
}

// Run polls Docker for service information until Stop() is called.
func (locator *Locator) Run() {
	if locator.done == nil {
		locator.done = make(chan struct{})
	}

	services := locator.load()
	if locator.diff(nil, services) {
		locator.Cache.Clear()
	}

	pollInterval := locator.PollInterval
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}

	for {
		select {
		case <-time.After(pollInterval):
			s := locator.load()
			if locator.diff(services, s) {
				locator.Cache.Clear()
			}
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
		locator.Logger.Println(err)
	}

	return new
}

func (locator *Locator) diff(old []ServiceInfo, new []ServiceInfo) bool {
	diff := false

	for _, info := range old {
		log := true
		for _, other := range new {
			if info.Equal(other) {
				log = false
				break
			}
		}

		if log {
			diff = true
			locator.Logger.Printf(
				"Removed route from '%s' to '%s' (%s)",
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
			diff = true
			locator.Logger.Printf(
				"Added route from '%s' to '%s' (%s)",
				info.Matcher.Pattern,
				info.Name,
				info.Endpoint.Description,
			)
		}
	}

	return diff
}
