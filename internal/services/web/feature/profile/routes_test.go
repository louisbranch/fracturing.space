package profile

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeProfileService struct {
	called bool
}

func (f *fakeProfileService) HandleProfile(http.ResponseWriter, *http.Request) {
	f.called = true
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeProfileService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/app/profile", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !svc.called {
		t.Fatal("expected profile handler to be called")
	}
}
