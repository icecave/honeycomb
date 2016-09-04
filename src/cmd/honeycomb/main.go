package main

import (
	"os"

	"github.com/icecave/honeycomb/src/di"
)

func main() {
	container := &di.Container{}
	defer container.Close()
	err := container.Server().Run()
	if err != nil {
		os.Exit(1)
	}
}
