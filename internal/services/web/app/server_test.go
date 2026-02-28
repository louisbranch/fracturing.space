package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

func TestComposeAppliesAuthToProtectedModules(t *testing.T) {
	t.Parallel()

	protected := stubModule{
		id: "protected",
		mount: module.Mount{
			Prefix: "/app/protected/",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
		},
	}

	authRequired := func(r *http.Request) bool {
		return r.Header.Get("X-Allow") == "yes"
	}

	h, err := Compose(ComposeInput{
		AuthRequired:     authRequired,
		ProtectedModules: []module.Module{protected},
	})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/app/protected/a", nil)
	blockedRR := httptest.NewRecorder()
	h.ServeHTTP(blockedRR, blockedReq)
	if blockedRR.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", blockedRR.Code, http.StatusFound)
	}
	if got := blockedRR.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/app/protected/a", nil)
	allowedReq.Header.Set("X-Allow", "yes")
	allowedRR := httptest.NewRecorder()
	h.ServeHTTP(allowedRR, allowedReq)
	if allowedRR.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", allowedRR.Code, http.StatusNoContent)
	}
}
