package htmx

import (
	"bytes"
	"html"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// ResponseHeaderKey is the HTMX request header used to detect partial updates.
const ResponseHeaderKey = "HX-Request"

// responseBuffer captures component rendering for HTMX responses.
type responseBuffer struct {
	header      http.Header
	statusCode  int
	body        bytes.Buffer
	headerWrote bool
}

func (w *responseBuffer) Header() http.Header {
	return w.header
}

func (w *responseBuffer) WriteHeader(status int) {
	if w.headerWrote {
		return
	}
	w.headerWrote = true
	w.statusCode = status
}

func (w *responseBuffer) Write(body []byte) (int, error) {
	return w.body.Write(body)
}

// IsHTMXRequest reports whether the request was initiated by HTMX.
func IsHTMXRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(r.Header.Get(ResponseHeaderKey), "true")
}

// TitleTag formats an escaped `<title>` element.
func TitleTag(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return "<title>" + html.EscapeString(title) + "</title>"
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func addHTMXTitleIfMissing(responseBody []byte, title string) []byte {
	bodyLower := strings.ToLower(string(responseBody))
	if strings.Contains(bodyLower, "<title") {
		return responseBody
	}
	if strings.TrimSpace(title) == "" {
		return responseBody
	}
	return append([]byte(title), responseBody...)
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Set-Cookie") {
			for _, value := range values {
				dst.Add(key, value)
			}
			continue
		}
		// Single-valued headers should not accumulate duplicates when copied from
		// a temporary response buffer.
		for _, value := range values {
			dst.Set(key, value)
		}
	}
}

// RenderPage renders a page for normal or HTMX requests.
//
// fragment is used for HTMX responses while full is used for non-HTMX responses.
// If fragment is nil, full is used for both paths.
func RenderPage(w http.ResponseWriter, r *http.Request, fragment templ.Component, full templ.Component, htmxTitle string) {
	if IsHTMXRequest(r) {
		target := fragment
		captureFromFull := full != nil
		if captureFromFull {
			target = full
		}
		if target == nil {
			return
		}
		capture := newResponseBuffer()
		templ.Handler(target).ServeHTTP(capture, r)

		body := capture.body.Bytes()
		if captureFromFull {
			if mainContent, ok := extractMainContent(body); ok {
				body = mainContent
			}
		}

		body = addHTMXTitleIfMissing(body, htmxTitle)
		copyHeaders(w.Header(), capture.Header())
		if !capture.headerWrote {
			capture.statusCode = http.StatusOK
		}
		if capture.statusCode != http.StatusOK {
			w.WriteHeader(capture.statusCode)
		}
		_, _ = w.Write(body)
		return
	}

	if full == nil {
		full = fragment
	}
	if full == nil {
		return
	}
	templ.Handler(full).ServeHTTP(w, r)
}

func extractMainContent(body []byte) ([]byte, bool) {
	start := bytes.Index(body, []byte("<main"))
	if start < 0 {
		return nil, false
	}
	openClose := bytes.Index(body[start:], []byte(">"))
	if openClose < 0 {
		return nil, false
	}
	contentStart := start + openClose + 1
	end := bytes.Index(body[contentStart:], []byte("</main>"))
	if end < 0 {
		return nil, false
	}
	return body[contentStart : contentStart+end], true
}
