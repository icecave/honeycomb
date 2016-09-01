package frontend

import (
	"fmt"

	"github.com/quipo/statsd"
)

type Metrics interface {
	StartRequest(ctx *requestContext)
	EndRequest(ctx *requestContext)
}

type StatsDMetrics struct {
	Client statsd.Statsd
}

func (metrics *StatsDMetrics) StartRequest(ctx *requestContext) {
	if ctx.IsWebSocket {
		metrics.Client.Incr("websocket.requests", 1)
	} else {
		metrics.Client.Incr(fmt.Sprintf(
			"http.requests.%s",
			ctx.Request.Method,
		), 1)
	}
}

func (metrics *StatsDMetrics) EndRequest(ctx *requestContext) {
	if ctx.IsWebSocket {
		metrics.Client.PrecisionTiming("websocket.ttfb", ctx.Timer.TimeToFirstByte())
		metrics.Client.PrecisionTiming("websocket.ttlb", ctx.Timer.TimeToLastByte())
	} else {
		metrics.Client.Incr(fmt.Sprintf(
			"http.responses.%s.%d",
			ctx.Request.Method,
			ctx.Writer.StatusCode,
		), 2)

		metrics.Client.PrecisionTiming("http.ttfb", ctx.Timer.TimeToFirstByte())
		metrics.Client.PrecisionTiming("http.ttlb", ctx.Timer.TimeToLastByte())
	}
}
