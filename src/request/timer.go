package request

import "time"

// Timer captures time offsets for key events during the life-cycle of an HTTP
// request.
type Timer struct {
	StartedAt       time.Time
	TimeToFirstByte float32
	TimeToLastByte  float32
}

// Start the timer.
func (timer *Timer) Start() {
	timer.StartedAt = time.Now()
}

// FirstByteSent records the time offset to the first byte.
func (timer *Timer) FirstByteSent() {
	duration := time.Now().Sub(timer.StartedAt)
	timer.TimeToFirstByte = float32(duration) / float32(time.Millisecond)
}

// LastByteSent records the time offset to the last byte.
func (timer *Timer) LastByteSent() {
	duration := time.Now().Sub(timer.StartedAt)
	timer.TimeToLastByte = float32(duration) / float32(time.Millisecond)
}
