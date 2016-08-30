package frontend

import "time"

type requestTimer struct {
	receivedAt  *time.Time
	respondedAt *time.Time
	completedAt *time.Time
}

func (timer *requestTimer) MarkReceived() {
	now := time.Now()
	timer.receivedAt = &now
}

func (timer *requestTimer) MarkResponded() {
	now := time.Now()
	timer.respondedAt = &now
}

func (timer *requestTimer) MarkCompleted() {
	now := time.Now()
	timer.completedAt = &now
}

func (timer *requestTimer) TimeToFirstByte() time.Duration {
	return timer.respondedAt.Sub(*timer.receivedAt)
}

func (timer *requestTimer) TotalTime() time.Duration {
	return timer.completedAt.Sub(*timer.receivedAt)
}

func (timer *requestTimer) HasResponded() bool {
	return timer.respondedAt != nil
}

func (timer *requestTimer) IsComplete() bool {
	return timer.completedAt != nil
}
