package di

import (
	"log"
	"os"
)

// Logger returns the application-wide logger.
func (con *Container) Logger() *log.Logger {
	return con.get(
		"logger",
		func() (interface{}, error) {
			return log.New(
				os.Stdout,
				"",
				log.LstdFlags,
			), nil
		},
		nil,
	).(*log.Logger)
}
