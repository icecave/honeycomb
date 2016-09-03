package docker

import "github.com/icecave/honeycomb/src/backend"

// ServiceInfo meta-data and a reference to the docker service used as a back-end.
type ServiceInfo struct {
	Name     string
	Matcher  *backend.Matcher
	Endpoint *backend.Endpoint
}

func (info ServiceInfo) Equal(other ServiceInfo) bool {
	return info.Name == other.Name &&
		*info.Matcher == *other.Matcher &&
		*info.Endpoint == *other.Endpoint
}
