package health

import (
	"context"

	"github.com/docker/docker/client"
)

// SwarmChecker is a checker that checks if the Docker connection is handled by
// a swarm manager.
type SwarmChecker struct {
	Client client.APIClient
}

// Check returns information about the health of the HTTPS server.
func (checker *SwarmChecker) Check() Status {
	if _, err := checker.Client.SwarmInspect(context.Background()); err != nil {
		return Status{false, err.Error()}
	}

	return Status{
		true,
		"The server is connected to a Docker swarm manager.",
	}
}
