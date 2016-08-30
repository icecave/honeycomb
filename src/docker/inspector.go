package docker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types/swarm"
)

// inspector extracts endpoint information from Docker services.
type inspector struct {
	client client.APIClient
}

type domainMatcher func(string) bool

func (in *inspector) DomainMatcher(
	_ context.Context,
	svc *swarm.Service,
) (domainMatcher, string, error) {
	value := svc.Spec.Labels[domainLabel]

	if value == "" {
		return nil, "", fmt.Errorf("'%s' label is missing or empty", domainLabel)
	} else if prefix := strings.TrimSuffix(value, ".*"); prefix != value {
		if isDomainName(prefix) {
			prefix = strings.ToLower(prefix) + "."
			return func(s string) bool {
				return strings.HasPrefix(
					strings.ToLower(s),
					prefix,
				)
			}, value, nil
		}
	} else if suffix := strings.TrimPrefix(value, "*."); suffix != value {
		if isDomainName(suffix) {
			suffix = "." + strings.ToLower(suffix)
			return func(s string) bool {
				return strings.HasSuffix(
					strings.ToLower(s),
					suffix,
				)
			}, value, nil
		}
	} else if isDomainName(value) {
		return func(s string) bool {
			return strings.EqualFold(s, value)
		}, value, nil
	}

	return nil, "", fmt.Errorf(
		"'%s' label value (%s) is not a valid domain name",
		domainLabel,
		value,
	)
}

func (in *inspector) IsTLS(
	_ context.Context,
	port int,
	service *swarm.Service,
) (bool, error) {
	if value, ok := service.Spec.Labels[isTLSLabel]; ok {
		switch value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf(
				"'%s' label must be either 'true' or 'false'",
				isTLSLabel,
			)
		}
	}

	switch port {
	case 443, 8443:
		return true, nil
	default:
		return false, nil
	}
}

func (in *inspector) Port(
	ctx context.Context,
	svc *swarm.Service,
) (int, error) {
	// There is a label to indicate the port, use it ...
	if value, ok := svc.Spec.Labels[portLabel]; ok {
		return net.LookupPort("tcp", value)
	}

	// Otherwise, look up the image to see if there is a single exposed port ...
	image, _, err := in.client.ImageInspectWithRaw(
		ctx,
		svc.Spec.TaskTemplate.ContainerSpec.Image,
	)
	if err != nil {
		return 0, err
	}

	// Search the exposed ports to find a single TCP port ...
	port := 0
	for p := range image.Config.ExposedPorts {
		if p.Proto() != "tcp" {
			continue
		} else if port == 0 {
			port = p.Int()
		} else {
			return 0, fmt.Errorf(
				"image '%s' exposes multiple TCP ports, add '%s' label to choose",
				image.Parent,
				portLabel,
			)
		}
	}

	if port == 0 {
		return 0, fmt.Errorf(
			"image '%s' does not expose any TCP ports",
			image.Parent,
		)
	}

	return port, nil
}
