package frontend

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/icecave/honeycomb/src/backend"
	"github.com/icecave/honeycomb/src/proxy"
)

type requestContext struct {
	Writer      proxy.ResponseWriter
	Request     *http.Request
	IsWebSocket bool
	Timer       requestTimer
	Endpoint    *backend.Endpoint
	Error       error
}

// String returns a representation of the request context suitable for logging.
func (ctx *requestContext) String() string {
	frontend := ""
	backend := "- -"
	statusCode := 0
	responseSize := "-"
	timeToFirstByte := "-"
	totalTime := "-"
	info := ""

	if ctx.Error != nil {
		info = ctx.Error.Error()
	}

	if ctx.IsWebSocket {
		frontend = "wss://" + ctx.Request.Host
		statusCode = http.StatusSwitchingProtocols
		responseSize = "-"
	} else {
		frontend = "https://" + ctx.Request.Host
		statusCode = ctx.Writer.StatusCode
		responseSize = strconv.Itoa(ctx.Writer.Size)
	}

	// @todo use endpoint.Name in the logs somewhere
	if ctx.Endpoint != nil {
		backend = fmt.Sprintf(
			"%s://%s %s",
			ctx.Endpoint.GetScheme(ctx.IsWebSocket),
			ctx.Endpoint.Address,
			ctx.Endpoint.Description,
		)
	}

	if ctx.Timer.HasResponded() {
		timeToFirstByte = ctx.Timer.TimeToFirstByte().String()

		if ctx.Timer.IsComplete() {
			totalTime = ctx.Timer.TimeToLastByte().String()
		} else if ctx.IsWebSocket && info == "" {
			info = "connection established"
		}
	}

	if info != "" {
		info = fmt.Sprintf(" (%s)", info)
	}

	return fmt.Sprintf(
		"%s %s %s \"%s %s %s\" %d %s %s %s%s",
		ctx.Request.RemoteAddr,
		frontend,
		backend,
		ctx.Request.Method,
		ctx.Request.URL,
		ctx.Request.Proto,
		statusCode,
		responseSize,
		timeToFirstByte,
		totalTime,
		info,
	)
}
