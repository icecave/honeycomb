package frontend

import (
	"fmt"
	"time"

	statsd "gopkg.in/alexcesaro/statsd.v2"
)

type Metrics interface {
	StartRequest(ctx *requestContext)
	EndRequest(ctx *requestContext)
}

type StatsDMetrics struct {
	Client *statsd.Client
}

func (metrics *StatsDMetrics) StartRequest(ctx *requestContext) {
	if ctx.IsWebSocket {
		metrics.Client.Increment("websocket.requests")
	} else {
		metrics.Client.Increment(fmt.Sprintf(
			"http.requests.%s",
			ctx.Request.Method,
		))
	}
}

func (metrics *StatsDMetrics) EndRequest(ctx *requestContext) {
	ttfb := int(ctx.Timer.TimeToFirstByte() / time.Millisecond)
	ttlb := int(ctx.Timer.TimeToLastByte() / time.Millisecond)

	if ctx.IsWebSocket {
		metrics.Client.Timing("websocket.ttfb", ttfb)
		metrics.Client.Timing("websocket.ttlb", ttlb)
	} else {
		metrics.Client.Increment(fmt.Sprintf(
			"http.responses.%s.%d",
			ctx.Request.Method,
			ctx.Writer.StatusCode,
		))

		metrics.Client.Timing("http.ttfb", ttfb)
		metrics.Client.Timing("http.ttlb", ttlb)
	}
}
