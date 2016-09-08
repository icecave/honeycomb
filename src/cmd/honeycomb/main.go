package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/icecave/honeycomb/src/di"
	"github.com/icecave/honeycomb/src/frontend/health"
)

func main() {
	defer di.Container.Close()

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

	if *check {
		checker := di.Container.Get("docker.health-checker").(health.Checker)
		status := checker.Check()
		fmt.Println(status.Message)
		if !status.IsHealthy {
			os.Exit(1)
		}
	} else {
		logger := di.Container.Get("logger").(*log.Logger)
		server := di.Container.Get("frontend.server").(*http.Server)
		logger.Printf("Listening on %s", server.Addr)
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Fatalln(err)
		}
	}
}
