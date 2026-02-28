package sessioncookie

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func TestRead(t *testing.T) {
	t.Parallel()

	if _, ok := Read(nil); ok {
		t.Fatalf("expected nil request to have no session cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, ok := Read(req); ok {
		t.Fatalf("expected missing cookie")
	}

	req.AddCookie(&http.Cookie{Name: Name, Value: "  ws-1  "})
	value, ok := Read(req)
	if !ok {
		t.Fatalf("expected cookie to be present")
	}
	if value != "ws-1" {
		t.Fatalf("value = %q, want %q", value, "ws-1")
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()

	secureReq := httptest.NewRequest(http.MethodGet, "https://app.example.test", nil)
	secureRR := httptest.NewRecorder()
	Write(secureRR, secureReq, "ws-1")
	secureCookie, err := http.ParseSetCookie(secureRR.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if secureCookie.Name != Name {
		t.Fatalf("cookie name = %q, want %q", secureCookie.Name, Name)
	}
	if secureCookie.Value != "ws-1" {
		t.Fatalf("cookie value = %q, want %q", secureCookie.Value, "ws-1")
	}
	if !secureCookie.Secure {
		t.Fatalf("expected secure cookie for https request")
	}

	httpReq := httptest.NewRequest(http.MethodGet, "http://app.example.test", nil)
	httpRR := httptest.NewRecorder()
	Write(httpRR, httpReq, "ws-1")
	httpCookie, err := http.ParseSetCookie(httpRR.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if httpCookie.Secure {
		t.Fatalf("expected non-secure cookie for http request")
	}

	policyReq := httptest.NewRequest(http.MethodGet, "http://app.example.test", nil)
	policyReq.Header.Set("X-Forwarded-Proto", "https")
	policyRR := httptest.NewRecorder()
	WriteWithPolicy(policyRR, policyReq, "ws-1", requestmeta.SchemePolicy{TrustForwardedProto: true})
	policyCookie, err := http.ParseSetCookie(policyRR.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if !policyCookie.Secure {
		t.Fatalf("expected secure cookie when trusted policy is enabled")
	}
}

func TestClear(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://app.example.test", nil)
	rr := httptest.NewRecorder()
	Clear(rr, req)
	cookie, err := http.ParseSetCookie(rr.Header().Get("Set-Cookie"))
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if cookie.Name != Name {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, Name)
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("cookie max-age = %d, want < 0", cookie.MaxAge)
	}
}
