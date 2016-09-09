package backend

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/icecave/honeycomb/src/name"
	"github.com/icecave/honeycomb/src/statuspage"
)

// Router is a proxy.Router that uses a locator to endpoints.
type Router struct {
	Locator Locator
}

// Route updates upstreamURL and upstreamHeaders as appropriate for the
// upstream server, based on request.
func (router *Router) Route(
	request *http.Request,
	isWebSocketRequest bool,
	upstreamURL *url.URL,
	upstreamHeaders http.Header,
) (string, error) {
	serverName, err := name.FromHTTP(request)
	if err != nil {
		return "", err
	}

	endpoint := router.Locator.Locate(request.Context(), serverName)
	if endpoint == nil {
		return "", statuspage.Error{
			Inner:      errors.New("could not locate backend"),
			StatusCode: http.StatusNotFound,
		}
	}

	upstreamURL.Host = endpoint.Address

	if isWebSocketRequest {
		if endpoint.IsTLS {
			upstreamURL.Scheme = "wss"
		} else {
			upstreamURL.Scheme = "ws"
		}
	} else {
		if endpoint.IsTLS {
			upstreamURL.Scheme = "https"
		} else {
			upstreamURL.Scheme = "http"
		}
	}

	return endpoint.Description, nil
}
