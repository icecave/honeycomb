package health

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	proxyproto "github.com/pires/go-proxyproto"
)

// HTTPChecker is a checker that connects to the HTTP server to check its status.
type HTTPChecker struct {
	Address      string
	Client       *http.Client
	ProxySupport bool
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
		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.Dial(network, addr)
			if err != nil {
				return nil, err
			}

			header := proxyproto.Header{
				Command: proxyproto.LOCAL,
				Version: 2,
			}
			_, err = header.WriteTo(conn)
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
		if !checker.ProxySupport {
			dialContext = nil
		}
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: dialContext,
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

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Status{false, err.Error()}
	}

	return Status{
		200 <= response.StatusCode && response.StatusCode <= 299,
		string(content),
	}
}
