package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/icecave/honeycomb/cmd"
	"github.com/icecave/honeycomb/docker/health"
	proxyproto "github.com/pires/go-proxyproto"
)

func main() {
	config := cmd.GetConfigFromEnvironment()

	checker := health.HTTPChecker{
		Address: ":" + config.Port,
		Client:  checkerHTTPClientProvider(config),
	}

	status := checker.Check()
	fmt.Println(status.Message)
	if !status.IsHealthy {
		os.Exit(1)
	}
}

func checkerHTTPClientProvider(config *cmd.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if config.ProxyProtocol {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
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
	}
	return &http.Client{
		Transport: transport,
		Timeout:   config.CheckTimeout,
	}
}
