package docker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/swarm"
	"github.com/icecave/honeycomb/src/backend"
)

// ServiceInspector inspects a Docker swarm service to produce information about
// an endpoint.
type ServiceInspector struct {
	Client client.APIClient
}

// Inspect attempts to produce an endpoint from the given Docker service.
func (inspector *ServiceInspector) Inspect(
	ctx context.Context,
	service *swarm.Service,
) (*backend.Endpoint, error) {
	port, err := inspector.port(ctx, service)
	if err != nil {
		return nil, err
	}

	isTLS, err := inspector.isTLS(service, port)
	if err != nil {
		return nil, err
	}

	return &backend.Endpoint{
		Description: service.Spec.TaskTemplate.ContainerSpec.Image,
		Address:     net.JoinHostPort(service.Spec.Name, port),
		IsTLS:       isTLS,
	}, nil
}

func (inspector *ServiceInspector) isTLS(
	service *swarm.Service,
	port string,
) (bool, error) {
	if value, ok := service.Spec.Labels[isTLSLabel]; ok {
		switch value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf(
				"invalid '%s' label (%s), expected true or false",
				isTLSLabel,
				value,
			)
		}
	}

	numeric, _ := net.LookupPort("tcp", port)
	switch numeric {
	case 443, 8443:
		return true, nil
	default:
		return false, nil
	}
}

func (inspector *ServiceInspector) port(
	ctx context.Context,
	service *swarm.Service,
) (string, error) {
	// Trust whatever is in the port label if it's present ...
	if value, ok := service.Spec.Labels[portLabel]; ok {
		_, err := net.LookupPort("tcp", value)

		if err != nil {
			return "", fmt.Errorf(
				"invalid '%s' label (%s), expected port name or number",
				portLabel,
				value,
			)
		}

		return value, nil
	}

	ports, err := inspector.exposedPorts(ctx, service)
	if err != nil {
		return "", err
	} else if len(ports) == 0 {
		return "", fmt.Errorf(
			"'%s' image does not expose any TCP ports",
			service.Spec.TaskTemplate.ContainerSpec.Image,
		)
	} else if len(ports) > 1 {
		return "", fmt.Errorf(
			"'%s' image exposes multiple TCP ports (%s), add a '%s' label to the service to select one",
			service.Spec.TaskTemplate.ContainerSpec.Image,
			strings.Join(ports, ", "),
			portLabel,
		)
	}

	return ports[0], nil
}

func (inspector *ServiceInspector) exposedPorts(
	ctx context.Context,
	service *swarm.Service,
) ([]string, error) {
	image, _, err := inspector.Client.ImageInspectWithRaw(
		ctx,
		service.Spec.TaskTemplate.ContainerSpec.Image,
	)
	if err != nil {
		return nil, err
	}

	var ports []string

	for p := range image.Config.ExposedPorts {
		if p.Proto() == "tcp" {
			ports = append(
				ports,
				p.Port(),
			)
		}
	}

	return ports, nil
}
