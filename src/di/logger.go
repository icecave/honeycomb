package di

import (
	"log"
	"os"

	"github.com/icecave/honeycomb/src/di/container"
)

func init() {
	Container.Define("logger", func(d *container.Definer) (interface{}, error) {
		return log.New(
			os.Stdout,
			"",
			log.LstdFlags,
		), nil
	})
}
