package di

import "github.com/icecave/honeycomb/src/frontend/health"

// HealthChecker returns the health-checker that is to be used to query the
// server's health.
func (con *Container) HealthChecker() health.Checker {
	return con.get(
		"docker.health-checker",
		func() (interface{}, error) {
			return &health.Client{Address: con.BindAddress()}, nil
		},
		nil,
	).(health.Checker)
}
