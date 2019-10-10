package docker

import (
	"github.com/icecave/honeycomb/backend"
	"github.com/icecave/honeycomb/name"
)

// ServiceInfo meta-data and a reference to the docker service used as a back-end.
type ServiceInfo struct {
	Name     string
	Matcher  *name.Matcher
	Endpoint *backend.Endpoint
}

// Equal checks if two ServiceInfo structs represent the same service.
func (info ServiceInfo) Equal(other ServiceInfo) bool {
	return info.Name == other.Name &&
		*info.Matcher == *other.Matcher &&
		*info.Endpoint == *other.Endpoint
}
