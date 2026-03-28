package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadRouteParamReturnsTrimmedValue(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("notificationID", " notification-1 ")

	got, ok := ReadRouteParam(req, "notificationID")
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got != "notification-1" {
		t.Fatalf("value = %q, want %q", got, "notification-1")
	}
}

func TestReadRouteParamRejectsMissingOrBlankValues(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got, ok := ReadRouteParam(req, "missing"); ok || got != "" {
		t.Fatalf("ReadRouteParam() = (%q, %t), want empty false", got, ok)
	}

	req.SetPathValue("missing", "   ")
	if got, ok := ReadRouteParam(req, "missing"); ok || got != "" {
		t.Fatalf("ReadRouteParam() = (%q, %t), want empty false", got, ok)
	}
}

func TestWithRequiredRouteParamCallsOnMissingForMissingValue(t *testing.T) {
	t.Parallel()

	called := false
	handler := WithRequiredRouteParam("credentialID", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNotFound)
	}, func(http.ResponseWriter, *http.Request, string) {
		t.Fatal("delegate should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("onMissing was not called")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestWithRequiredRouteParamDelegatesResolvedValue(t *testing.T) {
	t.Parallel()

	var got string
	handler := WithRequiredRouteParam("username", func(http.ResponseWriter, *http.Request) {
		t.Fatal("onMissing should not be called")
	}, func(_ http.ResponseWriter, _ *http.Request, value string) {
		got = value
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("username", "  louis  ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got != "louis" {
		t.Fatalf("value = %q, want %q", got, "louis")
	}
}
