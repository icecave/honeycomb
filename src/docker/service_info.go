package docker

import (
	"github.com/docker/engine-api/types/swarm"
	"github.com/icecave/honeycomb/src/backend"
)

// ServiceInfo meta-data and a reference to the docker service used as a back-end.
type ServiceInfo struct {
	DockerService *swarm.Service
	Matcher       *backend.Matcher
	Endpoint      *backend.Endpoint
}
