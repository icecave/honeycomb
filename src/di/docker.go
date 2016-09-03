package di

import (
	"os"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/docker"
)

// DockerClient returns the docker client used to access the swarm.
func (con *Container) DockerClient() client.APIClient {
	return con.get(
		"docker.client",
		func() (interface{}, error) {
			return client.NewEnvClient()
		},
		nil,
	).(client.APIClient)
}

// ServiceLoader returns the service loader used to load Docker services.
func (con *Container) ServiceLoader() *docker.ServiceLoader {
	return con.get(
		"docker.service-loader",
		func() (interface{}, error) {
			return &docker.ServiceLoader{
				Client:    con.DockerClient(),
				Inspector: con.ServiceInspector(),
				Logger:    con.Logger(),
			}, nil
		},
		nil,
	).(*docker.ServiceLoader)
}

// ServiceInspector returns the service inspector used to create endpoints from
// Docker services.
func (con *Container) ServiceInspector() *docker.ServiceInspector {
	return con.get(
		"docker.service-inspector",
		func() (interface{}, error) {
			return &docker.ServiceInspector{
				Client: con.DockerClient(),
			}, nil
		},
		nil,
	).(*docker.ServiceInspector)
}

// DockerPollInterval returns the interval at which the Docker service list is
// queried.
func (con *Container) DockerPollInterval() time.Duration {
	return con.get(
		"docker.interval",
		func() (interface{}, error) {
			if interval := os.Getenv("DOCKER_INTERVAL"); interval != "" {
				return time.ParseDuration(interval)
			}

			return docker.DefaultPollInterval, nil
		},
		nil,
	).(time.Duration)
}

// DockerLocator returns the back-end locator used to resolve server names to
// Docker services.
func (con *Container) DockerLocator() backend.Locator {
	return con.get(
		"docker.locator",
		func() (interface{}, error) {
			locator := docker.NewLocator(
				con.DockerPollInterval(),
				con.ServiceLoader(),
				con.Logger(),
			)

			go locator.Run()

			return locator, nil
		},
		func(value interface{}) error {
			value.(*docker.Locator).Stop()
			return nil
		},
	).(backend.Locator)
}
