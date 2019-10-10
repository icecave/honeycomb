package docker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/icecave/honeycomb/backend"
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

	tlsMode, err := inspector.tlsMode(service, port)
	if err != nil {
		return nil, err
	}

	return &backend.Endpoint{
		Description: inspector.description(service),
		Address:     net.JoinHostPort(service.Spec.Name, port),
		TLSMode:     tlsMode,
	}, nil
}

func (inspector *ServiceInspector) description(service *swarm.Service) string {
	if value, ok := service.Spec.Labels[descriptionLabel]; ok {
		return value
	}

	image := service.Spec.TaskTemplate.ContainerSpec.Image
	ref, err := reference.Parse(image)
	if err != nil {
		return image
	}

	if r, ok := ref.(reference.NamedTagged); ok {
		return fmt.Sprintf("%s:%s", r.Name(), r.Tag())
	}

	return ref.String()
}

func (inspector *ServiceInspector) tlsMode(
	service *swarm.Service,
	port string,
) (backend.TLSMode, error) {
	if value, ok := service.Spec.Labels[tlsLabel]; ok {
		switch value {
		case "true", "enabled":
			return backend.TLSEnabled, nil
		case "false", "disabled":
			return backend.TLSDisabled, nil
		case "insecure":
			return backend.TLSInsecure, nil
		default:
			return backend.TLSDisabled, fmt.Errorf(
				"invalid '%s' label (%s), expected 'enabled', 'disabled' or 'insecure'",
				tlsLabel,
				value,
			)
		}
	}

	numeric, _ := net.LookupPort("tcp", port)
	switch numeric {
	case 443, 8443:
		return backend.TLSEnabled, nil
	default:
		return backend.TLSDisabled, nil
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
