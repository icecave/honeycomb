package statuspage

import (
	"bytes"
	htmlTemplate "html/template"
	"net/http"
	textTemplate "text/template"

	"github.com/golang/gddo/httputil/header"
	"github.com/icecave/honeycomb/artifacts/assets"
)

// TemplateWriter writes status pages in HTML or plain-text format using a
// template.
type TemplateWriter struct {
	HTMLTemplate *htmlTemplate.Template
	TextTemplate *textTemplate.Template
}

// TemplateContext holds the data needed to render a status page.
type TemplateContext struct {
	Code    int
	Text    string
	Message string
}

// Write outputs a status page for statusCode to writer, in response to request.
func (wr *TemplateWriter) Write(
	writer http.ResponseWriter,
	request *http.Request,
	statusCode int,
) (bodySize int64, err error) {
	return wr.WriteMessage(
		writer,
		request,
		statusCode,
		StatusMessage(statusCode),
	)
}

// WriteMessage outputs an HTTP status page for statusCode to writer, in
// response to request, including a custom message.
func (wr *TemplateWriter) WriteMessage(
	writer http.ResponseWriter,
	request *http.Request,
	statusCode int,
	message string,
) (int64, error) {
	var buf bytes.Buffer
	var contentType string
	context := TemplateContext{
		statusCode,
		http.StatusText(statusCode),
		message,
	}

	if useHTML(request) {
		tmpl := wr.HTMLTemplate
		if tmpl == nil {
			tmpl = defaultHTMLTemplate
		}

		if err := tmpl.Execute(&buf, context); err == nil {
			contentType = "text/html"
		}
	}

	if contentType == "" {
		tmpl := wr.TextTemplate
		if tmpl == nil {
			tmpl = defaultTextTemplate
		}
		contentType = "text/plain"
		buf.Reset()
		tmpl.Execute(&buf, context)
	}

	writer.Header().Add("Content-Type", contentType+"; charset=utf-8")
	writer.WriteHeader(statusCode)
	return buf.WriteTo(writer)
}

// WriteError outputs an appropriate HTTP status page for the given error to
// writer, in response to request.
func (wr *TemplateWriter) WriteError(
	writer http.ResponseWriter,
	request *http.Request,
	statusErr error,
) (statusCode int, bodySize int64, err error) {
	if e, ok := statusErr.(Error); ok {
		statusCode = e.StatusCode
		if e.Message != "" {
			bodySize, err = wr.WriteMessage(
				writer,
				request,
				statusCode,
				e.Message,
			)
			return
		}
	} else {
		statusCode = http.StatusInternalServerError
	}

	bodySize, err = wr.Write(writer, request, statusCode)
	return
}

var defaultHTMLTemplate *htmlTemplate.Template
var defaultTextTemplate *textTemplate.Template

func init() {
	defaultHTMLTemplate = htmlTemplate.Must(
		htmlTemplate.New("status-page").Parse(assets.STATUS_PAGE_HTML),
	)
	defaultTextTemplate = textTemplate.Must(
		textTemplate.New("status-page").Parse(assets.STATUS_PAGE_TXT),
	)
}

func useHTML(request *http.Request) bool {
	htmlQ := -1.0
	textQ := 0.0

	for _, spec := range header.ParseAccept(request.Header, "Accept") {
		if spec.Value == "text/html" || spec.Value == "application/xhtml+xml" {
			if spec.Q > htmlQ {
				htmlQ = spec.Q
			}
		} else if spec.Value == "text/plain" || spec.Value == "*.*" {
			if spec.Q > textQ {
				textQ = spec.Q
			}
		}
	}

	return htmlQ > textQ
}
