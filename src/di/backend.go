package di

import (
	"log"
	"time"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/di/container"
	"github.com/icecave/honeycomb/src/docker"
)

func init() {
	Container.Define("backend.locator", func(d *container.Definer) (interface{}, error) {
		dockerLocator := docker.NewLocator(
			d.Get("DOCKER_POLL_INTERVAL").(time.Duration),
			d.Get("docker.service-loader").(*docker.ServiceLoader),
			d.Get("logger").(*log.Logger),
		)

		go dockerLocator.Run()
		d.Defer(dockerLocator.Stop)

		return backend.AggregateLocator{
			backend.StaticLocator{}.With(
				"static.*",
				&backend.Endpoint{
					Description: "local-echo-server",
					Address:     "localhost:8080",
				},
			),
			dockerLocator,
		}, nil
	})
}
