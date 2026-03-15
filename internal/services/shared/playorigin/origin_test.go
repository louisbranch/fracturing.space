package playorigin

import (
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

func TestPlayURLUsesSubdomainForNonLoopbackHosts(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.com:8080/app/campaigns/c1/game", nil)
	got := PlayURL(req, requestmeta.SchemePolicy{}, "8094", "/campaigns/c1")
	if got != "http://play.example.com:8080/campaigns/c1" {
		t.Fatalf("PlayURL() = %q, want %q", got, "http://play.example.com:8080/campaigns/c1")
	}
}

func TestPlayURLUsesFallbackPortForLoopbackHosts(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://localhost:8080/app/campaigns/c1/game", nil)
	got := PlayURL(req, requestmeta.SchemePolicy{}, "8094", "/campaigns/c1")
	if got != "http://localhost:8094/campaigns/c1" {
		t.Fatalf("PlayURL() = %q, want %q", got, "http://localhost:8094/campaigns/c1")
	}
}

func TestWebURLUsesFallbackPortForLoopbackHosts(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://play.localhost:8094/campaigns/c1", nil)
	got := WebURL(req, requestmeta.SchemePolicy{}, "8080", "/app/campaigns/c1")
	if got != "http://localhost:8080/app/campaigns/c1" {
		t.Fatalf("WebURL() = %q, want %q", got, "http://localhost:8080/app/campaigns/c1")
	}
}
