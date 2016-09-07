package request

import "time"

// Timer captures time offsets for key events during the life-cycle of an HTTP
// request.
type Timer struct {
	StartedAt       time.Time
	TimeToFirstByte time.Duration
	TimeToLastByte  time.Duration
}

// Start the timer.
func (timer *Timer) Start() {
	timer.StartedAt = time.Now()
}

// FirstByteSent records the time offset to the first byte.
func (timer *Timer) FirstByteSent() {
	timer.TimeToFirstByte = time.Now().Sub(timer.StartedAt)
}

// LastByteSent records the time offset to the last byte.
func (timer *Timer) LastByteSent() {
	timer.TimeToLastByte = time.Now().Sub(timer.StartedAt)
}
