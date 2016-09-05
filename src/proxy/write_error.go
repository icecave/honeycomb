package proxy

import (
	"html/template"
	"io"
	"net/http"

	"github.com/icecave/honeycomb/src/assets"
)

var errorTemplate *template.Template
var statusMessages = map[int]string{
	// 4xx
	http.StatusBadRequest:                  "Your browser has sent a malformed request.",
	http.StatusUnauthorized:                "You must be authenticated to use this service.",
	http.StatusForbidden:                   "You do not have access to this service.",
	http.StatusNotFound:                    "The page you've requested could not be found.",
	http.StatusNotAcceptable:               "The content of this page is not accepted by your browser.",
	http.StatusProxyAuthRequired:           "You must be authenticated to use this service.",
	http.StatusRequestTimeout:              "Your browser did not send a request in a timely manner.",
	http.StatusRequestEntityTooLarge:       "Your browser has sent a request that's too large to process.",
	http.StatusRequestURITooLong:           "Your browser has sent a request with a URI that's too large to process.",
	http.StatusUpgradeRequired:             "Maybe you're trying to access a WebSocket server?",
	http.StatusTooManyRequests:             "Your request has been rate-limited, please descrease the number of requests.",
	http.StatusRequestHeaderFieldsTooLarge: "Your browser has sent a request header that is too large to process.",
	http.StatusUnavailableForLegalReasons:  "Your request has been denied for legal reasons.",

	// 5xx
	http.StatusNotImplemented:          "The feature you've requested is not supported.",
	http.StatusBadGateway:              "The service you've requested is temporarily unavailable, please try again.",
	http.StatusServiceUnavailable:      "The service you've requested does not exist.", // we send 503 when there are no backends
	http.StatusGatewayTimeout:          "The service you've requested did not respond in a timely manner, please try again.",
	http.StatusHTTPVersionNotSupported: "Your browser's HTTP version is not supported.",
}

func init() {
	errorTemplate = template.New("error")
	errorTemplate.Parse(assets.Asset_error_page_html)
}

// WriteError writes an HTML error page to the response.
func WriteError(
	writer http.ResponseWriter,
	statusCode int,
) {
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(statusCode)

	data := struct {
		Code    int
		Text    string
		Message string
	}{
		statusCode,
		http.StatusText(statusCode),
		StatusMessage(statusCode),
	}

	err := errorTemplate.Execute(writer, data)
	if err != nil {
		io.WriteString(writer, data.Text)
	}
}

// StatusMessage returns a short, human-readable description of the given HTTP
// status code.
func StatusMessage(statusCode int) string {
	message := statusMessages[statusCode]
	if message == "" {
		if 400 <= statusCode && statusCode <= 599 {
			return "We're sorry, something went wrong!"
		}

		return "That's all we know."
	}

	return message
}
