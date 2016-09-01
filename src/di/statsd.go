package di

import (
	"os"
	"time"

	"github.com/quipo/statsd"
)

// StatsDAddress returns the network address of the statsd server.
func (con *Container) StatsDAddress() string {
	return os.Getenv("STATSD_ADDRESS")
}

// StatsDPrefix returns the prefix to use for all statistics.
func (con *Container) StatsDPrefix() string {
	if prefix := os.Getenv("STATSD_PREFIX"); prefix != "" {
		return prefix
	}

	return "honeycomb."
}

// StatsDInterval returns the interval at which stats are flushed.
func (con *Container) StatsDInterval() time.Duration {
	return con.get(
		"statsd.interval",
		func() (interface{}, error) {
			if interval := os.Getenv("STATSD_INTERVAL"); interval != "" {
				return time.ParseDuration(interval)
			}

			return time.Duration(0), nil
		},
		nil,
	).(time.Duration)
}

// StatsDClient returns the statsd client used to send metrics.
func (con *Container) StatsDClient() statsd.Statsd {
	return con.get(
		"statsd.client",
		func() (interface{}, error) {
			client := statsd.NewStatsdClient(
				con.StatsDAddress(),
				con.StatsDPrefix(),
			)

			err := client.CreateSocket()
			if err != nil {
				return nil, err
			}

			interval := con.StatsDInterval()
			if interval == 0 {
				return client, nil
			}

			return statsd.NewStatsdBuffer(interval, client), nil
		},
		func(value interface{}) error {
			value.(statsd.Statsd).Close()
			return nil
		},
	).(statsd.Statsd)
}
