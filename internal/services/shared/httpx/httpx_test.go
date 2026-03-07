package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChainAppliesMiddlewareInOrder(t *testing.T) {
	var order []string
	mw := func(label string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, label)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { order = append(order, "handler") }),
		mw("first"),
		mw("second"),
	)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if len(order) != 3 || order[0] != "first" || order[1] != "second" || order[2] != "handler" {
		t.Fatalf("middleware order = %v, want [first second handler]", order)
	}
}

func TestChainNilHandler(t *testing.T) {
	handler := Chain(nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("nil handler status = %d, want 404", rec.Code)
	}
}

func TestRequestIDSetsHeader(t *testing.T) {
	handler := RequestID("test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rid := rec.Header().Get("X-Request-ID"); rid == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestRequestIDEchoesExisting(t *testing.T) {
	handler := RequestID("test")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "existing-id" {
		t.Fatalf("X-Request-ID = %q, want existing-id", got)
	}
}

func TestRecoverPanicReturns500(t *testing.T) {
	handler := RecoverPanic()(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic recovery status = %d, want 500", rec.Code)
	}
}

func TestRecoverPanicNoEffect(t *testing.T) {
	handler := RecoverPanic()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("no-panic status = %d, want 200", rec.Code)
	}
}
