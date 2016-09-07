package di

import (
	"log"
	"os"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/icecave/honeycomb/src/di/container"
	"github.com/icecave/honeycomb/src/docker"
)

func init() {
	Container.Define("DOCKER_POLL_INTERVAL", func(d *container.Definer) (interface{}, error) {
		if interval := os.Getenv("DOCKER_POLL_INTERVAL"); interval != "" {
			return time.ParseDuration(interval)
		}

		return docker.DefaultPollInterval, nil
	})

	Container.Define("docker.client", func(d *container.Definer) (interface{}, error) {
		return client.NewEnvClient()
	})

	Container.Define("docker.service-loader", func(d *container.Definer) (interface{}, error) {
		return &docker.ServiceLoader{
			Client:    d.Get("docker.client").(client.APIClient),
			Inspector: d.Get("docker.service-inspector").(*docker.ServiceInspector),
			Logger:    d.Get("logger").(*log.Logger),
		}, nil
	})

	Container.Define("docker.service-inspector", func(d *container.Definer) (interface{}, error) {
		return &docker.ServiceInspector{
			Client: d.Get("docker.client").(client.APIClient),
		}, nil
	})
}
