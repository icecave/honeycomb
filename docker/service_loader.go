package docker

import (
	"context"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/name"
)

// ServiceLoader loads information about Docker services that are marked as
// back-ends.
type ServiceLoader struct {
	Client    client.APIClient
	Inspector *ServiceInspector
	Logger    *log.Logger
}

// Load returns information about Docker services that are marked as back-ends.
func (loader *ServiceLoader) Load(
	ctx context.Context,
) ([]ServiceInfo, error) {
	options := types.ServiceListOptions{}

	services, err := loader.Client.ServiceList(ctx, options)
	if err != nil {
		return nil, err
	}

	var result []ServiceInfo

	for _, service := range services {
		matchers := loader.matchers(service)
		if len(matchers) == 0 {
			continue
		}

		endpoint, err := loader.Inspector.Inspect(ctx, &service)
		if err != nil {
			loader.Logger.Printf(
				"Can not route to '%s' (%s), %s",
				service.Spec.Name,
				service.Spec.TaskTemplate.ContainerSpec.Image,
				err,
			)
			continue
		}

		for _, matcher := range matchers {
			result = append(result, ServiceInfo{
				Name:     service.Spec.Name,
				Matcher:  matcher,
				Endpoint: endpoint,
			})
		}
	}

	return result, nil
}

func (loader *ServiceLoader) matchers(service swarm.Service) []*name.Matcher {
	var result []*name.Matcher

	for key, value := range service.Spec.Annotations.Labels {
		if key == matchLabel || strings.HasPrefix(key, matchLabel+".") {
			matcher, err := name.NewMatcher(value)

			if err != nil {
				loader.Logger.Printf(
					"Can not route to '%s' (%s) via '%s', %s",
					service.Spec.Name,
					service.Spec.TaskTemplate.ContainerSpec.Image,
					value,
					err,
				)
				continue
			}

			result = append(result, matcher)
		}
	}

	return result
}
