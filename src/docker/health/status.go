package health

import "fmt"

// Status is the result of a health-check.
type Status struct {
	IsHealthy bool
	Message   string
}

func (status Status) String() string {
	var s string
	if status.IsHealthy {
		s = "passed"
	} else {
		s = "failed"
	}

	return fmt.Sprintf(
		"Health-check %s: %s",
		s,
		status.Message,
	)
}
