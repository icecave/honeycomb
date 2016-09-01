package di

import (
	"os"
	"time"

	statsd "gopkg.in/alexcesaro/statsd.v2"
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

	return "honeycomb"
}

// StatsDInterval returns the interval at which stats are flushed.
func (con *Container) StatsDInterval() time.Duration {
	return con.get(
		"statsd.interval",
		func() (interface{}, error) {
			if interval := os.Getenv("STATSD_INTERVAL"); interval != "" {
				return time.ParseDuration(interval)
			}

			return time.Duration(100 * time.Millisecond), nil
		},
		nil,
	).(time.Duration)
}

// StatsDClient returns the statsd client used to send metrics.
func (con *Container) StatsDClient() *statsd.Client {
	return con.get(
		"statsd.client",
		func() (interface{}, error) {
			// If there is an error, the client will be "muted", essentially
			// a no-op client.  @todo, use a client that can recover from failures
			client, _ := statsd.New(
				statsd.Address(con.StatsDAddress()),
				statsd.Prefix(con.StatsDPrefix()),
				statsd.FlushPeriod(con.StatsDInterval()),
			)

			return client, nil
		},
		func(value interface{}) error {
			value.(*statsd.Client).Close()
			return nil
		},
	).(*statsd.Client)
}
