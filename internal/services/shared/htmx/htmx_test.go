package htmx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testComponent struct {
	body       string
	statusCode int
	headerKey  string
	headerVal  string
}

func (c testComponent) Render(_ context.Context, w io.Writer) error {
	if c.statusCode > 0 {
		if rw, ok := w.(http.ResponseWriter); ok {
			if c.headerKey != "" {
				rw.Header().Set(c.headerKey, c.headerVal)
			}
			rw.WriteHeader(c.statusCode)
		}
	}
	_, err := w.Write([]byte(c.body))
	return err
}

func TestIsHTMXRequest(t *testing.T) {
	t.Run("missing_request_is_not_htmx", func(t *testing.T) {
		t.Parallel()
		if got := IsHTMXRequest(nil); got {
			t.Fatalf("IsHTMXRequest(nil) = true, want false")
		}
	})

	t.Run("true_request_is_htmx", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.Header.Set(ResponseHeaderKey, "true")
		if got := IsHTMXRequest(r); !got {
			t.Fatalf("IsHTMXRequest(request) = false, want true")
		}
	})
}

func TestTitleTag(t *testing.T) {
	t.Parallel()
	got := TitleTag(`Campaign <Admin>`)
	want := "<title>Campaign &lt;Admin&gt;</title>"
	if got != want {
		t.Fatalf("TitleTag(...) = %q, want %q", got, want)
	}
}

func TestRenderPageForNonHTMXUsesFullRender(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	fragment := testComponent{body: "<div>fragment</div>"}
	full := testComponent{body: "<html><body>full</body></html>"}

	RenderPage(w, r, fragment, full, TitleTag("Provided"))
	if got := w.Body.String(); got != "<html><body>full</body></html>" {
		t.Fatalf("rendered body = %q, want full page body", got)
	}
}

func TestRenderPageForHTMXInjectsMissingTitle(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set(ResponseHeaderKey, "true")
	w := httptest.NewRecorder()

	fragment := testComponent{body: "<main>fragment</main>"}
	RenderPage(w, r, fragment, nil, TitleTag("Fragment Page"))

	got := w.Body.String()
	if !strings.HasPrefix(got, "<title>Fragment Page</title>") {
		t.Fatalf("expected injected title prefix in HTMX response, got %q", got)
	}
	if !strings.HasSuffix(got, "<main>fragment</main>") {
		t.Fatalf("expected original fragment to remain, got %q", got)
	}
}

func TestRenderPageForHTMXPreservesExistingTitle(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set(ResponseHeaderKey, "true")
	w := httptest.NewRecorder()

	fragment := testComponent{body: "<title>Already Set</title><main>fragment</main>"}
	RenderPage(w, r, fragment, nil, TitleTag("Injected Title"))

	got := w.Body.String()
	if !strings.Contains(got, "<title>Already Set</title>") {
		t.Fatalf("expected existing title preserved, got %q", got)
	}
	if !strings.Contains(got, "<main>fragment</main>") {
		t.Fatalf("expected fragment body preserved, got %q", got)
	}
}

func TestRenderPageForHTMXNoInjectedTitleWhenMissing(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set(ResponseHeaderKey, "true")
	w := httptest.NewRecorder()

	fragment := testComponent{
		body: "<main>fragment</main>",
	}
	RenderPage(w, r, fragment, nil, "")

	got := w.Body.String()
	if got != "<main>fragment</main>" {
		t.Fatalf("rendered body = %q, want %q", got, "<main>fragment</main>")
	}
}

func TestCopyHeadersUsesSingleValueSemanticsForNonSetCookie(t *testing.T) {
	t.Parallel()
	dst := http.Header{}
	src := http.Header{}
	src.Add("Content-Type", "text/plain")
	src.Add("Content-Type", "text/html; charset=utf-8")
	src.Add("Set-Cookie", "id=1")
	src.Add("Set-Cookie", "token=abc")

	copyHeaders(dst, src)

	contentType := dst.Values("Content-Type")
	if len(contentType) != 1 {
		t.Fatalf("expected one Content-Type value, got %v", contentType)
	}
	if got := contentType[0]; got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	cookies := dst.Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("expected two Set-Cookie values, got %v", cookies)
	}
}
