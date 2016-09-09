package frontend

import "net/http"

// ConditionalHandler is an interface for http.Handler instances that optionally
// intercept an incoming request.
type ConditionalHandler interface {
	http.Handler

	// CanHandle returns true if request can be served by this handler.
	CanHandle(*http.Request) bool
}
