package publicprofile

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	called bool
}

func (f *fakeService) HandlePublicProfile(http.ResponseWriter, *http.Request) {
	f.called = true
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/u/alice", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if !svc.called {
		t.Fatal("expected public profile handler to be called")
	}
}
