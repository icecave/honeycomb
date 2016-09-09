package health

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

// HTTPChecker is a checker that connects to the HTTP server to check its status.
type HTTPChecker struct {
	Address string
	Client  *http.Client
}

// Check returns information about the health of the HTTPS server.
func (checker *HTTPChecker) Check() Status {
	host, port, err := net.SplitHostPort(checker.Address)
	if err != nil {
		return Status{false, err.Error()}
	} else if host == "" {
		host = requestHost
	}

	client := checker.Client
	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	var url url.URL
	url.Scheme = "https"
	url.Host = net.JoinHostPort(host, port)
	url.Path = requestPath

	response, err := client.Get(url.String())
	if err != nil {
		return Status{false, err.Error()}
	}

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Status{false, err.Error()}
	}

	return Status{
		200 <= response.StatusCode && response.StatusCode <= 299,
		string(content),
	}
}
