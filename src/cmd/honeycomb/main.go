package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/icecave/honeycomb/src/di"
)

func main() {
	check := flag.Bool(
		"check",
		false,
		"Perform a health-check instead of starting the server.",
	)

	flag.Parse()

	if len(flag.Args()) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	container := &di.Container{}
	defer container.Close()

	success := false
	if *check {
		status := container.HealthChecker().Check()
		fmt.Println(status.Message)
		success = status.IsHealthy
	} else {
		success = container.Server().Run() == nil
	}

	if !success {
		os.Exit(1)
	}
}
