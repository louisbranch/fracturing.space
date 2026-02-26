package httpx

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestChainAppliesMiddlewareInOrder(t *testing.T) {
	t.Parallel()

	called := ""
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called += "1"
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called += "2"
			next.ServeHTTP(w, r)
		})
	}

	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called += "h"
		w.WriteHeader(http.StatusNoContent)
	}), mw1, mw2)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if called != "12h" {
		t.Fatalf("call order = %q, want %q", called, "12h")
	}
}

func TestRequireMethodRejectsUnexpectedMethod(t *testing.T) {
	t.Parallel()

	h := RequireMethod(http.MethodGet)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMethodNotAllowedWritesAllowHeaderAndStatus(t *testing.T) {
	t.Parallel()

	h := MethodNotAllowed(http.MethodPost)
	req := httptest.NewRequest(http.MethodGet, "/app/settings/ai-keys/cred-1/revoke", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestRequestIDAddsHeaderWhenMissing(t *testing.T) {
	t.Parallel()

	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			t.Fatalf("expected request header to include request id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected response to include request id")
	}
}

func TestRecoverPanicReturnsInternalServerError(t *testing.T) {
	t.Parallel()

	h := RecoverPanic()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestRecoverPanicLogsRequestContext(t *testing.T) {
	t.Parallel()

	prevWriter := log.Writer()
	defer log.SetOutput(prevWriter)
	var buffer bytes.Buffer
	log.SetOutput(&buffer)

	h := RecoverPanic()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Request-ID", "req-123")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	logLine := buffer.String()
	for _, marker := range []string{"panic recovered", "path=/panic", "request_id=req-123"} {
		if !strings.Contains(logLine, marker) {
			t.Fatalf("panic log missing marker %q: %q", marker, logLine)
		}
	}
}

func TestWriteJSONSetsContentTypeAndBody(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	err := WriteJSON(rr, http.StatusOK, struct {
		Value string `json:"value"`
	}{Value: "ok"})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}
	if body := rr.Body.String(); !strings.Contains(body, "\"value\":\"ok\"") {
		t.Fatalf("body = %q, want encoded json", body)
	}
}

func TestWriteErrorUsesTypedStatus(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WriteError(rr, apperrors.E(apperrors.KindUnauthorized, "missing session"))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestWriteErrorDefaultsInternalError(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WriteError(rr, errors.New("boom"))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestIsHTMXRequestDetectsHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if IsHTMXRequest(req) {
		t.Fatalf("expected non-htmx request")
	}
	req.Header.Set("HX-Request", "true")
	if !IsHTMXRequest(req) {
		t.Fatalf("expected htmx request")
	}
}

func TestWriteHTMLSetsContentType(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	err := WriteHTML(rr, http.StatusCreated, "<div>ok</div>")
	if err != nil {
		t.Fatalf("WriteHTML() error = %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
}

func TestWriteHXRedirectSetsHeader(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WriteHXRedirect(rr, "/app/invites")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != "/app/invites" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/app/invites")
	}
}

func TestWriteRedirectUsesLocationForNonHTMX(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1/sessions/start", nil)
	rr := httptest.NewRecorder()
	WriteRedirect(rr, req, "/app/campaigns/c1/sessions")
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/app/campaigns/c1/sessions" {
		t.Fatalf("Location = %q, want %q", got, "/app/campaigns/c1/sessions")
	}
	if got := rr.Header().Get("HX-Redirect"); got != "" {
		t.Fatalf("HX-Redirect = %q, want empty", got)
	}
}

func TestWriteRedirectUsesHXRedirectForHTMX(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/c1/sessions/start", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	WriteRedirect(rr, req, "/app/campaigns/c1/sessions")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != "/app/campaigns/c1/sessions" {
		t.Fatalf("HX-Redirect = %q, want %q", got, "/app/campaigns/c1/sessions")
	}
}

func TestWriteRedirectHandlesNilRequest(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	WriteRedirect(rr, nil, "/app/campaigns/c1/sessions")
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/app/campaigns/c1/sessions" {
		t.Fatalf("Location = %q, want %q", got, "/app/campaigns/c1/sessions")
	}
}

func TestWriteErrorNilAndNilWriterSafety(t *testing.T) {
	t.Parallel()

	WriteError(nil, errors.New("ignored"))

	rr := httptest.NewRecorder()
	WriteError(rr, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWriteJSONAndWriteHTMLRequireWriter(t *testing.T) {
	t.Parallel()

	if err := WriteJSON(nil, http.StatusOK, map[string]string{"ok": "true"}); err == nil {
		t.Fatalf("expected WriteJSON(nil) error")
	}
	if err := WriteHTML(nil, http.StatusOK, "ok"); err == nil {
		t.Fatalf("expected WriteHTML(nil) error")
	}
}

func TestIsHTMXRequestHandlesNilRequest(t *testing.T) {
	t.Parallel()

	if IsHTMXRequest(nil) {
		t.Fatalf("expected nil request to be non-HTMX")
	}
}

func TestRequestIDPreservesIncomingHeader(t *testing.T) {
	t.Parallel()

	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Request-ID"); got != "req-123" {
			t.Fatalf("request id = %q, want %q", got, "req-123")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("response request id = %q, want %q", got, "req-123")
	}
}

func TestChainHandlesNilHandlerAndMiddleware(t *testing.T) {
	t.Parallel()

	h := Chain(nil, nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/no-route", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestWriteHXRedirectNilWriterSafety(t *testing.T) {
	t.Parallel()

	WriteHXRedirect(nil, "/ignored")
}

func TestWriteRedirectNilWriterSafety(t *testing.T) {
	t.Parallel()

	WriteRedirect(nil, nil, "/ignored")
}
