package frontend

import "time"

type requestTimer struct {
	ReceivedAt  time.Time
	RespondedAt time.Time
	CompletedAt time.Time
}

func (timer *requestTimer) MarkReceived() {
	timer.ReceivedAt = time.Now()
}

func (timer *requestTimer) MarkResponded() {
	timer.RespondedAt = time.Now()
}

func (timer *requestTimer) MarkCompleted() {
	timer.CompletedAt = time.Now()
}

func (timer *requestTimer) TimeToFirstByte() time.Duration {
	return timer.RespondedAt.Sub(timer.ReceivedAt)
}

func (timer *requestTimer) TransmissionTime() time.Duration {
	return timer.CompletedAt.Sub(timer.RespondedAt)
}

func (timer *requestTimer) TotalTime() time.Duration {
	return timer.CompletedAt.Sub(timer.ReceivedAt)
}
