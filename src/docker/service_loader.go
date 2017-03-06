package docker

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/src/name"
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
	filter := filters.NewArgs()
	filter.Add("label", matchLabel)
	options := types.ServiceListOptions{Filters: filter}

	services, err := loader.Client.ServiceList(ctx, options)
	if err != nil {
		return nil, err
	}

	var result []ServiceInfo

	for _, service := range services {
		var err error
		info := ServiceInfo{Name: service.Spec.Name}

		info.Matcher, err = name.NewMatcher(
			service.Spec.Annotations.Labels[matchLabel],
		)

		if err == nil {
			info.Endpoint, err = loader.Inspector.Inspect(ctx, &service)
			if err == nil {
				result = append(result, info)
				continue
			}
		}

		loader.Logger.Printf(
			"Can not route to '%s' (%s), %s",
			service.Spec.Name,
			service.Spec.TaskTemplate.ContainerSpec.Image,
			err,
		)
	}

	return result, nil
}
