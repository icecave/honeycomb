package static

import (
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/name"
)

// FromEnv returns static locators configured by environment variables.
func FromEnv(logger *log.Logger) (Locator, error) {
	locator, err := fromEnv(os.Environ())
	if err != nil {
		return nil, err
	}

	for _, p := range locator {
		logger.Printf(
			"Added static route from '%s' to '%s' (%s)",
			p.Matcher.Pattern,
			p.Endpoint.Address,
			p.Endpoint.Description,
		)
	}

	return locator, nil
}

func fromEnv(env []string) (Locator, error) {
	var locator Locator

	for _, e := range env {
		groups := routePattern.FindStringSubmatch(e)
		if len(groups) == 0 {
			continue
		}

		matcher, err := name.NewMatcher(groups[matcherIndex])
		if err != nil {
			return nil, err
		}

		u, err := url.Parse(groups[addressIndex])
		if err != nil {
			return nil, err
		}

		isTLS := u.Scheme == "https" || u.Scheme == "wss"

		if port := u.Port(); port == "" {
			if isTLS {
				u.Host += ":443"
			} else {
				u.Host += ":80"
			}
		}

		endpoint := &backend.Endpoint{
			Description: groups[tagIndex],
			Address:     u.Host,
			IsTLS:       isTLS,
		}

		if groups[descriptionIndex] != "" {
			endpoint.Description = groups[descriptionIndex]
		}

		locator = append(locator, matcherEndpointPair{matcher, endpoint})
	}

	return locator, nil
}

const (
	tagIndex = iota + 1
	matcherIndex
	addressIndex
	descriptionIndex
)

var routePattern = regexp.MustCompile(`^ROUTE_([^\s]+)=([^\s]+) ([^\s]+)(?: (.+))?$`)
