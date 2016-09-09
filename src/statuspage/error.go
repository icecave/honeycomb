package statuspage

// Error wraps another error to include the appropriate HTTP status code to send
// as a result of this error.
type Error struct {
	Inner      error
	StatusCode int
	Message    string
}

func (err Error) Error() string {
	return err.Inner.Error()
}
