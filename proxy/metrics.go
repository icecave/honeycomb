package proxy

import "time"

// Metrics stores basic measuresments for a request.
type Metrics struct {
	// BytesIn is the total number of bytes received for this request.
	// Includes websocket frames, but not HTTP headers.
	BytesIn int64

	// BytesOut is the total number of bytes sent in response to this
	// request. Includes websocket frames, but not HTTP headers.
	BytesOut int64

	StartedAt       time.Time
	TimeToFirstByte float64
	TimeToLastByte  float64
}

// Start the timer.
func (metrics *Metrics) Start() {
	metrics.StartedAt = time.Now()
}

// FirstByteSent records the time offset to the first byte.
func (metrics *Metrics) FirstByteSent() {
	duration := time.Now().Sub(metrics.StartedAt)
	metrics.TimeToFirstByte = float64(duration) / float64(time.Millisecond)
}

// IsFirstByteSent returns true if the first byte has been sent.
func (metrics *Metrics) IsFirstByteSent() bool {
	return metrics.TimeToFirstByte > 0
}

// LastByteSent records the time offset to the last byte.
func (metrics *Metrics) LastByteSent() {
	duration := time.Now().Sub(metrics.StartedAt)
	metrics.TimeToLastByte = float64(duration) / float64(time.Millisecond)
}

// IsLastByteSent returns true if the last byte has been sent.
func (metrics *Metrics) IsLastByteSent() bool {
	return metrics.TimeToLastByte > 0
}
