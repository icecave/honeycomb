package main

import (
	"fmt"
	"os"

	"github.com/icecave/honeycomb/src/cmd"
	"github.com/icecave/honeycomb/src/docker/health"
)

func main() {
	config := cmd.GetConfigFromEnvironment()

	checker := health.HTTPChecker{
		Address:      ":" + config.Port,
		ProxySupport: config.ProxySupport,
	}

	status := checker.Check()
	fmt.Println(status.Message)
	if !status.IsHealthy {
		os.Exit(1)
	}
}
