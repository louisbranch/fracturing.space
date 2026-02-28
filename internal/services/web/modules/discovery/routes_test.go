package discovery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestRegisterRoutesHandlesNilMux(t *testing.T) {
	t.Parallel()

	registerRoutes(nil, newHandlers(publichandler.NewBase()))
}

func TestRegisterRoutesDiscoveryMethodContract(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	registerRoutes(mux, newHandlers(publichandler.NewBase()))

	getReq := httptest.NewRequest(http.MethodGet, routepath.DiscoverPrefix, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", getRR.Code, http.StatusOK)
	}
	if getRR.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", getRR.Header().Get("Content-Type"), "text/html; charset=utf-8")
	}
	if body := getRR.Body.String(); !strings.Contains(body, "discover-root") {
		t.Fatalf("body missing discovery marker: %q", body)
	}

	headReq := httptest.NewRequest(http.MethodHead, routepath.DiscoverPrefix, nil)
	headRR := httptest.NewRecorder()
	mux.ServeHTTP(headRR, headReq)
	if headRR.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", headRR.Code, http.StatusOK)
	}

	postReq := httptest.NewRequest(http.MethodPost, routepath.DiscoverPrefix, nil)
	postRR := httptest.NewRecorder()
	mux.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", postRR.Code, http.StatusMethodNotAllowed)
	}
	if got := postRR.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q, want %q", got, "GET, HEAD")
	}
}
