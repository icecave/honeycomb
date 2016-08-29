package main

import "github.com/icecave/honeycomb/src/di"

func main() {
	container := &di.Container{}
	defer container.Close()
	err := container.Server().Run()
	if err != nil {
		container.Logger().Fatalln(err)
	}
}
