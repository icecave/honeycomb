package transaction

// State indicates a request's progress through its life-cycle.
type State int

const (
	// StateReceived is the initial state of the request.
	StateReceived State = iota

	// StateResponded means that the response headers have been sent.
	StateResponded

	// StateClosed means that the request has been handled and is complete.
	StateClosed
)
