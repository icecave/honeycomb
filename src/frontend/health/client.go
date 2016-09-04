package health

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

// Client is a checker that connects to the HTTP server to check its status.
type Client struct {
	Address string
}

// Check returns information about the health of the HTTPS server.
func (client *Client) Check() Status {
	host, port, err := net.SplitHostPort(client.Address)
	if host == "" {
		host = "localhost"
	}

	var url url.URL
	url.Scheme = "https"
	url.Host = net.JoinHostPort(host, port)
	url.Path = healthCheckPath

	transport := http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // @todo take CA cert and verify
		},
	}

	httpClient := http.Client{
		Transport: &transport,
	}

	response, err := httpClient.Get(url.String())
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
