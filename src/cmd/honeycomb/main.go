package main

import (
	"fmt"

	"github.com/icecave/honeycomb/src/di"
)

func main() {
	c := &di.Container{}
	cli := c.DockerClient()

	fmt.Println(cli)
}
